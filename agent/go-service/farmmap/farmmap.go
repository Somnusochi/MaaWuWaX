// Package farmmap implements star-path world farming inspired by ok-ww.
package farmmap

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/minicv"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/mouse"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/walk"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

type pointBox struct {
	X int
	Y int
	W int
	H int
}

func (b pointBox) center() (float64, float64) {
	return float64(b.X) + float64(b.W)/2, float64(b.Y) + float64(b.H)/2
}

func (b pointBox) distance(o pointBox) float64 {
	x1, y1 := b.center()
	x2, y2 := o.center()
	return math.Hypot(x2-x1, y2-y1)
}

func (b pointBox) scale(f float64) pointBox {
	cx, cy := b.center()
	w := int(float64(b.W) * f)
	h := int(float64(b.H) * f)
	if w < 40 {
		w = 40
	}
	if h < 40 {
		h = 40
	}
	return pointBox{X: int(cx) - w/2, Y: int(cy) - h/2, W: w, H: h}
}

type stateData struct {
	bigMap       *image.RGBA
	stars        []pointBox
	myBox        pointBox
	miniMapBox   pointBox
	lastDistance float64
	stuckIndex   int
	done         bool
	tracker      *MotionTracker
}

var farmState = struct {
	sync.Mutex
	state stateData
}{}

type loadPathParam struct {
	MaxStepDistance int `json:"max_step_distance"`
	MinStars        int `json:"min_stars"`
}

type walkStepParam struct {
	ReachDistance int `json:"reach_distance"`
	SearchRadius  int `json:"search_radius"`
	WalkMs        int `json:"walk_ms"`
}

type LoadPathAction struct{}

var _ maa.CustomActionRunner = &LoadPathAction{}

// ResetTrackerAction resets the motion tracker and farm state.
// Useful after teleporting or zone changes.
type ResetTrackerAction struct{}

var _ maa.CustomActionRunner = &ResetTrackerAction{}

func (a *ResetTrackerAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	farmState.Lock()
	if farmState.state.tracker != nil {
		farmState.state.tracker.ForceLost()
	}
	farmState.state.stuckIndex = 0
	farmState.state.lastDistance = 0
	farmState.Unlock()
	log.Info().Str("component", "FarmMapResetTracker").Msg("tracker and movement state reset")
	return true
}

func (a *LoadPathAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := loadPathParam{MaxStepDistance: 144, MinStars: 3}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "FarmMapLoadPath").Msg("failed to parse param")
		}
	}

	screen, err := ctx.GetTasker().GetController().CacheImage()
	if err != nil {
		log.Warn().Err(err).Str("component", "FarmMapLoadPath").Msg("failed to cache image")
		return false
	}
	img := minicv.ImageConvertRGBA(screen)

	diamond, ok := findBestTemplate(ctx, img, "big_map_diamond.png", 0.7)
	if !ok {
		log.Error().Str("component", "FarmMapLoadPath").Msg("need a diamond marker as the starting point")
		return false
	}

	stars := findAllTemplate(ctx, img, "big_map_star.png", 0.7)
	path := sortPath(stars, diamond, float64(param.MaxStepDistance))
	path = append([]pointBox{diamond}, path...)
	if len(path) < param.MinStars {
		log.Error().
			Str("component", "FarmMapLoadPath").
			Int("stars", len(path)).
			Msg("need a path with at least 3 markers")
		return false
	}

	miniMap := detectMiniMap(ctx, img)
	myBox := diamond.scale(float64(miniMap.W) / math.Max(float64(diamond.W), 1) * 2)

	farmState.Lock()
	farmState.state = stateData{
		bigMap:     img,
		stars:      path,
		myBox:      clampBox(myBox, img.Bounds()),
		miniMapBox: miniMap,
		done:       false,
		tracker:    NewMotionTracker(),
	}
	farmState.Unlock()

	ctx.GetTasker().GetController().PostClickKey(keycode.MustCode("ESC")).Wait()
	time.Sleep(1200 * time.Millisecond)

	log.Info().
		Str("component", "FarmMapLoadPath").
		Int("markers", len(path)).
		Msg("loaded farm-map star path")
	return true
}

type PathDoneRecognition struct{}

var _ maa.CustomRecognitionRunner = &PathDoneRecognition{}

func (r *PathDoneRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	farmState.Lock()
	done := farmState.state.done || len(farmState.state.stars) == 0
	left := len(farmState.state.stars)
	farmState.Unlock()
	if !done {
		return nil, false
	}
	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: fmt.Sprintf(`{"done":true,"left":%d}`, left),
	}, true
}

type LocateRecognition struct{}

var _ maa.CustomRecognitionRunner = &LocateRecognition{}

func (r *LocateRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	farmState.Lock()
	state := farmState.state
	farmState.Unlock()
	if state.bigMap == nil || state.miniMapBox.W <= 0 || state.miniMapBox.H <= 0 {
		log.Warn().Str("component", "MapLocateRecognition").Msg("farm-map state is not initialized")
		return nil, false
	}

	img := minicv.ImageConvertRGBA(arg.Img)
	if img == nil {
		return nil, false
	}

	radius := 280
	if arg.CustomRecognitionParam != "" {
		var param struct {
			SearchRadius int `json:"search_radius"`
		}
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "MapLocateRecognition").Msg("failed to parse param")
		} else if param.SearchRadius > 0 {
			radius = param.SearchRadius
		}
	}

	loc, ok := findLocationInBigMap(state, img, radius)
	if !ok {
		log.Warn().Str("component", "MapLocateRecognition").Int("search_radius", radius).Msg("failed to locate minimap position")
		return nil, false
	}

	if state.tracker != nil {
		state.tracker.Update(loc)
	}
	state.myBox = loc.scale(1.3)
	saveState(state)

	payload, _ := sonic.Marshal(map[string]any{
		"status":        "success",
		"x":             loc.X,
		"y":             loc.Y,
		"width":         loc.W,
		"height":        loc.H,
		"search_radius": radius,
	})
	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{loc.X, loc.Y, loc.W, loc.H},
		Detail: string(payload),
	}, true
}

type WalkStepAction struct{}

var _ maa.CustomActionRunner = &WalkStepAction{}

func (a *WalkStepAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := walkStepParam{ReachDistance: 24, SearchRadius: 280, WalkMs: 850}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "FarmMapWalkStep").Msg("failed to parse param")
		}
	}

	farmState.Lock()
	state := farmState.state
	farmState.Unlock()
	if state.bigMap == nil || len(state.stars) == 0 {
		markDone()
		return true
	}

	screen, err := ctx.GetTasker().GetController().CacheImage()
	if err != nil {
		log.Warn().Err(err).Str("component", "FarmMapWalkStep").Msg("failed to cache image")
		return false
	}
	img := minicv.ImageConvertRGBA(screen)

	// Use motion tracker's predicted search rect for better tracking
	searchRadius := param.SearchRadius
	if state.tracker != nil && state.tracker.IsTracking(3) {
		predicted := state.tracker.PredictNextSearchRect(searchRadius)
		loc, ok := findLocationInBigMapPredicted(state, img, predicted)
		if ok {
			// Teleport detection
			if state.tracker.IsTeleport(loc) {
				log.Warn().Str("component", "FarmMapWalkStep").Msg("teleport detected, resetting tracker")
				state.tracker.ForceLost()
			}
			state.tracker.Update(loc)
			return walkTowardTarget(ctx, state, loc, param)
		}
		// Tracking failed: fall through to wider search
		state.tracker.MarkLost()
	}

	loc, ok := findLocationInBigMap(state, img, searchRadius)
	if !ok {
		loc = state.myBox
		log.Warn().Str("component", "FarmMapWalkStep").Msg("location match failed, using last known position")
	}

	if state.tracker != nil {
		state.tracker.Update(loc)
	}
	return walkTowardTarget(ctx, state, loc, param)
}

// walkTowardTarget encapsulates the star-reach / navigation logic.
func walkTowardTarget(ctx *maa.Context, state stateData, loc pointBox, param walkStepParam) bool {
	sort.Slice(state.stars, func(i, j int) bool {
		return loc.distance(state.stars[i]) < loc.distance(state.stars[j])
	})
	target := state.stars[0]
	distance := loc.distance(target)
	if distance <= float64(param.ReachDistance) {
		state.stars = state.stars[1:]
		state.lastDistance = 0
		state.stuckIndex = 0
		state.myBox = target.scale(1.3)
		if len(state.stars) == 0 {
			state.done = true
		}
		saveState(state)
		log.Info().
			Str("component", "FarmMapWalkStep").
			Int("left", len(state.stars)).
			Float64("distance", distance).
			Msg("reached star marker")
		return true
	}

	ctrl := ctx.GetTasker().GetController()
	if math.Abs(distance-state.lastDistance) < 1 {
		doStuckStep(ctrl, state.stuckIndex)
		state.stuckIndex++
	} else {
		angle := angleClockwise(loc, target)
		screen, _ := ctrl.CacheImage()
		screenImg := minicv.ImageConvertRGBA(screen)
		facing := detectFacingAngleEnhanced(screenImg, state.miniMapBox)
		turn := angleBetween(facing, angle)
		navigate(ctrl, turn, time.Duration(param.WalkMs)*time.Millisecond)
		state.stuckIndex = 0
		log.Debug().
			Str("component", "FarmMapWalkStep").
			Float64("distance", distance).
			Float64("target_angle", angle).
			Float64("facing", facing).
			Float64("turn", turn).
			Msg("walked toward star")
	}

	state.lastDistance = distance
	state.myBox = loc.scale(1.3)
	saveState(state)
	return true
}

func saveState(state stateData) {
	farmState.Lock()
	farmState.state = state
	farmState.Unlock()
}

func markDone() {
	farmState.Lock()
	farmState.state.done = true
	farmState.Unlock()
}

func findBestTemplate(ctx *maa.Context, img image.Image, template string, threshold float64) (pointBox, bool) {
	detail, err := ctx.RunRecognitionDirect(
		maa.RecognitionTypeTemplateMatch,
		&maa.TemplateMatchParam{
			Template:  []string{template},
			Threshold: []float64{threshold},
			OrderBy:   maa.TemplateMatchOrderByScore,
		},
		img,
	)
	if err != nil || detail == nil || !detail.Hit {
		return pointBox{}, false
	}
	return rectToBox(detail.Box), true
}

func findAllTemplate(ctx *maa.Context, img image.Image, template string, threshold float64) []pointBox {
	detail, err := ctx.RunRecognitionDirect(
		maa.RecognitionTypeTemplateMatch,
		&maa.TemplateMatchParam{
			Template:  []string{template},
			Threshold: []float64{threshold},
			OrderBy:   maa.TemplateMatchOrderByHorizontal,
		},
		img,
	)
	if err != nil || detail == nil || !detail.Hit || detail.Results == nil {
		return nil
	}
	results := detail.Results.Filtered
	if len(results) == 0 {
		results = detail.Results.All
	}
	boxes := make([]pointBox, 0, len(results))
	for _, result := range results {
		if item, ok := result.AsTemplateMatch(); ok {
			boxes = append(boxes, rectToBox(item.Box))
		}
	}
	return boxes
}

func detectMiniMap(ctx *maa.Context, img image.Image) pointBox {
	box, ok := findBestTemplate(ctx, img, "box_minimap.png", 0.6)
	if ok {
		return box
	}
	return pointBox{X: 1050, Y: 20, W: 200, H: 160}
}

func sortPath(points []pointBox, start pointBox, maxDistance float64) []pointBox {
	unvisited := append([]pointBox(nil), points...)
	path := make([]pointBox, 0, len(unvisited))
	current := start
	for len(unvisited) > 0 {
		best := -1
		bestDistance := math.MaxFloat64
		for i, candidate := range unvisited {
			d := current.distance(candidate)
			if maxDistance > 0 && d > maxDistance {
				continue
			}
			if d < bestDistance {
				best = i
				bestDistance = d
			}
		}
		if best < 0 {
			break
		}
		next := unvisited[best]
		path = append(path, next)
		unvisited = append(unvisited[:best], unvisited[best+1:]...)
		current = next
	}
	return path
}

func findLocationInBigMap(state stateData, current *image.RGBA, radius int) (pointBox, bool) {
	mini := cropEnhanced(current, state.miniMapBox)
	if mini == nil {
		return pointBox{}, false
	}
	search := expandBox(state.myBox, radius, state.bigMap.Bounds())
	best, ok := matchMiniMapEnhanced(state.bigMap, mini, search)
	if !ok {
		best, ok = matchMiniMapEnhanced(state.bigMap, mini, imageBoundsBox(state.bigMap.Bounds()))
	}
	return best, ok
}

// findLocationInBigMapPredicted uses a predicted search rect from the motion tracker.
func findLocationInBigMapPredicted(state stateData, current *image.RGBA, predicted pointBox) (pointBox, bool) {
	mini := cropEnhanced(current, state.miniMapBox)
	if mini == nil {
		return pointBox{}, false
	}
	// Try predicted region first, then widen
	best, ok := matchMiniMapEnhanced(state.bigMap, mini, clampBox(predicted, state.bigMap.Bounds()))
	if !ok {
		best, ok = matchMiniMapEnhanced(state.bigMap, mini, imageBoundsBox(state.bigMap.Bounds()))
	}
	return best, ok
}

func matchMiniMap(big *image.RGBA, mini *image.RGBA, search pointBox) (pointBox, bool) {
	return matchMiniMapEnhanced(big, mini, search)
}

// matchMiniMapEnhanced uses the improved minimap mask with UI filtering.
func matchMiniMapEnhanced(big *image.RGBA, mini *image.RGBA, search pointBox) (pointBox, bool) {
	w, h := mini.Rect.Dx(), mini.Rect.Dy()
	if w <= 0 || h <= 0 || w > big.Rect.Dx() || h > big.Rect.Dy() {
		return pointBox{}, false
	}

	// Use enhanced mask with UI filtering
	maskCfg := DefaultMinimapMaskConfig()
	miniMasked := ApplyMinimapMask(mini, maskCfg)

	const maskColor = 0x00FF00
	stats := minicv.GetImageStats(miniMasked)
	if stats.Std < 1e-12 {
		return pointBox{}, false
	}
	intArr := minicv.GetIntegralArray(big)
	x, y, score := minicv.MatchTemplateInAreaWithMask(big, intArr, miniMasked, stats, maskColor, [4]int{search.X, search.Y, search.W, search.H})
	if score < 0.2 {
		return pointBox{}, false
	}
	return pointBox{X: int(math.Round(x)), Y: int(math.Round(y)), W: w, H: h}, true
}

// cropEnhanced crops an image region and returns nil if the result is empty.
func cropEnhanced(img *image.RGBA, box pointBox) *image.RGBA {
	bounds := img.Bounds()
	box = clampBox(box, bounds)
	if box.W <= 0 || box.H <= 0 {
		return nil
	}
	return minicv.ImageCropRect(img, image.Rect(box.X, box.Y, box.X+box.W, box.Y+box.H))
}

func detectFacingAngle(img *image.RGBA, miniMap pointBox) float64 {
	return detectFacingAngleEnhanced(img, miniMap)
}

// detectFacingAngleEnhanced uses InferArrowAngle from tracker.go for
// fast, accurate moment-based direction detection instead of brute-force rotation.
func detectFacingAngleEnhanced(img *image.RGBA, miniMap pointBox) float64 {
	roi := cropEnhanced(img, miniMap)
	if roi == nil {
		return 0
	}
	angle := InferArrowAngle(roi)
	if angle < 0 {
		// Fallback to brute-force rotation matching
		return detectFacingAngleFallback(img, miniMap)
	}
	return angle
}

// detectFacingAngleFallback is the original brute-force rotation approach.
func detectFacingAngleFallback(img *image.RGBA, miniMap pointBox) float64 {
	template, err := loadImageAsset("arrow.png")
	if err != nil {
		log.Debug().Err(err).Str("component", "FarmMapWalkStep").Msg("arrow asset missing")
		return 0
	}
	roi := crop(img, miniMap)
	if roi == nil {
		return 0
	}
	bestAngle := 0.0
	bestScore := -1.0
	roiIntArr := minicv.GetIntegralArray(roi)
	for angle := 0; angle < 360; angle += 10 {
		rotated := minicv.ImageRotate(template, float64(angle))
		stats := minicv.GetImageStats(rotated)
		if stats.Std < 1e-12 {
			continue
		}
		_, _, score := minicv.MatchTemplateInArea(roi, roiIntArr, rotated, stats, [4]int{0, 0, roi.Rect.Dx(), roi.Rect.Dy()})
		if score > bestScore {
			bestScore = score
			bestAngle = float64(angle)
		}
	}
	return bestAngle
}

func navigate(ctrl *maa.Controller, turn float64, hold time.Duration) {
	mouse.MiddleClick(ctrl)
	switch {
	case turn > 135 || turn < -135:
		mouse.RotateCamera(ctrl, 420, 0)
		walk.Walk(ctrl, walk.Forward, hold)
	case turn > 45:
		mouse.RotateCamera(ctrl, 260, 0)
		walk.Walk(ctrl, walk.Forward, hold)
	case turn < -45:
		mouse.RotateCamera(ctrl, -260, 0)
		walk.Walk(ctrl, walk.Forward, hold)
	case turn > 10:
		ctrl.PostKeyDown(keycode.MustCode("D")).Wait()
		time.Sleep(120 * time.Millisecond)
		ctrl.PostKeyUp(keycode.MustCode("D")).Wait()
		walk.Walk(ctrl, walk.Forward, hold)
	case turn < -10:
		ctrl.PostKeyDown(keycode.MustCode("A")).Wait()
		time.Sleep(120 * time.Millisecond)
		ctrl.PostKeyUp(keycode.MustCode("A")).Wait()
		walk.Walk(ctrl, walk.Forward, hold)
	default:
		walk.Walk(ctrl, walk.Forward, hold)
	}
}

func doStuckStep(ctrl *maa.Controller, idx int) {
	steps := []struct {
		key string
		ms  int
	}{
		{"SPACE", 80},
		{"A", 700},
		{"D", 700},
		{"T", 80},
	}
	step := steps[idx%len(steps)]
	code := keycode.MustCode(step.key)
	ctrl.PostKeyDown(code).Wait()
	time.Sleep(time.Duration(step.ms) * time.Millisecond)
	ctrl.PostKeyUp(code).Wait()
	time.Sleep(250 * time.Millisecond)
	log.Info().Str("component", "FarmMapWalkStep").Str("key", step.key).Msg("stuck recovery step")
}

func angleClockwise(from pointBox, to pointBox) float64 {
	x1, y1 := from.center()
	x2, y2 := to.center()
	degree := math.Atan2(y2-y1, x2-x1) * 180 / math.Pi
	if degree < 0 {
		degree += 360
	}
	return degree
}

func angleBetween(current float64, target float64) float64 {
	turn := target - current
	for turn > 180 {
		turn -= 360
	}
	for turn < -180 {
		turn += 360
	}
	return turn
}

func loadImageAsset(name string) (*image.RGBA, error) {
	wd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(wd, "assets", "resource", "image", name),
		filepath.Join(wd, "..", "..", "assets", "resource", "image", name),
		filepath.Join(wd, "..", "assets", "resource", "image", name),
	}
	for _, candidate := range candidates {
		file, err := os.Open(candidate)
		if err != nil {
			continue
		}
		defer file.Close()
		img, err := png.Decode(file)
		if err != nil {
			return nil, err
		}
		return minicv.ImageConvertRGBA(img), nil
	}
	return nil, fmt.Errorf("asset %s not found", name)
}

func rectToBox(rect maa.Rect) pointBox {
	return pointBox{X: rect[0], Y: rect[1], W: rect[2], H: rect[3]}
}

func crop(img *image.RGBA, box pointBox) *image.RGBA {
	bounds := img.Bounds()
	box = clampBox(box, bounds)
	if box.W <= 0 || box.H <= 0 {
		return nil
	}
	return minicv.ImageCropRect(img, image.Rect(box.X, box.Y, box.X+box.W, box.Y+box.H))
}

func clampBox(box pointBox, bounds image.Rectangle) pointBox {
	x := max(box.X, bounds.Min.X)
	y := max(box.Y, bounds.Min.Y)
	w := min(box.X+box.W, bounds.Max.X) - x
	h := min(box.Y+box.H, bounds.Max.Y) - y
	return pointBox{X: x, Y: y, W: max(w, 0), H: max(h, 0)}
}

func expandBox(box pointBox, radius int, bounds image.Rectangle) pointBox {
	return clampBox(pointBox{
		X: box.X - radius,
		Y: box.Y - radius,
		W: box.W + radius*2,
		H: box.H + radius*2,
	}, bounds)
}

func imageBoundsBox(bounds image.Rectangle) pointBox {
	return pointBox{X: bounds.Min.X, Y: bounds.Min.Y, W: bounds.Dx(), H: bounds.Dy()}
}

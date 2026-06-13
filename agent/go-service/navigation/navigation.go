// Package navigation implements map navigation Custom Recognition for Wuthering Waves.
package navigation

import (
	"image"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/farmmap"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/minicv"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/mouse"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/walk"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// MinimapNavigateRecognition — performs a single forward step (W key).
// Used by EchoFarm and other navigation flows.
// ---------------------------------------------------------------------------

type MinimapNavigateRecognition struct{}

var _ maa.CustomRecognitionRunner = &MinimapNavigateRecognition{}

type minimapNavigateParam struct {
	TargetNode        string  `json:"target_node"`
	MinimapBox        [4]int `json:"minimap_box"`
	ForwardMs         int     `json:"forward_ms"`
	Sprint            bool    `json:"sprint"`
	CenterCamera      bool    `json:"center_camera"`
	StrongTurnDeg     float64 `json:"strong_turn_deg"`
	MediumTurnDeg     float64 `json:"medium_turn_deg"`
	SmallTurnDeg      float64 `json:"small_turn_deg"`
}

func (r *MinimapNavigateRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := defaultMinimapNavigateParam()
	if arg != nil && arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "MinimapNavigate").Msg("failed to parse param")
		}
	}
	param = normalizeMinimapNavigateParam(param)

	ctrl := ctx.GetTasker().GetController()
	if param.CenterCamera {
		mouse.MiddleClick(ctrl)
	}

	detail, err := ctx.RunRecognition(param.TargetNode, nil)
	if err == nil && detail != nil && detail.Hit {
		screen, cacheErr := ctrl.CacheImage()
		if cacheErr == nil {
			img := minicv.ImageConvertRGBA(screen)
			mini := cropMiniMap(img, param.MinimapBox)
			if mini != nil {
				facing := farmmap.InferArrowAngle(mini)
				if facing >= 0 {
					turn := angleDeltaDegrees(facing, targetAngleFromBox(detail.Box, param.MinimapBox))
					navigateMinimapTurn(ctrl, turn, time.Duration(param.ForwardMs)*time.Millisecond, param.Sprint, param)
					return &maa.CustomRecognitionResult{
						Box: detail.Box,
						Detail: `{"action":"navigate_minimap","turn":` +
							formatFloat(turn) + `,"facing":` + formatFloat(facing) + `}`,
					}, true
				}
			}
		}
	}

	if param.Sprint {
		walk.Sprint(ctrl, walk.Forward, time.Duration(param.ForwardMs)*time.Millisecond)
	} else {
		walk.Walk(ctrl, walk.Forward, time.Duration(param.ForwardMs)*time.Millisecond)
	}
	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"action":"step_forward","reason":"target_missing"}`,
	}, true
}

// ---------------------------------------------------------------------------
// DirectionWalkAction — walks in a WASD direction for a specified duration.
// Param: {"direction": "forward"|"back"|"left"|"right", "duration_ms": 3000}
// ---------------------------------------------------------------------------

type DirectionWalkAction struct{}

var _ maa.CustomActionRunner = &DirectionWalkAction{}

type walkParam struct {
	Direction  string `json:"direction"`
	DurationMs int    `json:"duration_ms"`
	Sprint     bool   `json:"sprint"`
}

func (a *DirectionWalkAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := walkParam{
		Direction:  "forward",
		DurationMs: 2000,
		Sprint:     false,
	}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "DirectionWalk").Msg("failed to parse param")
		}
	}

	ctrl := ctx.GetTasker().GetController()
	dir := parseDirection(param.Direction)
	dur := time.Duration(param.DurationMs) * time.Millisecond

	log.Debug().
		Str("component", "DirectionWalk").
		Str("direction", param.Direction).
		Int("duration_ms", param.DurationMs).
		Bool("sprint", param.Sprint).
		Msg("walking")

	if param.Sprint {
		walk.Sprint(ctrl, dir, dur)
	} else {
		walk.Walk(ctrl, dir, dur)
	}

	return true
}

func parseDirection(s string) walk.Direction {
	switch s {
	case "back":
		return walk.Back
	case "left":
		return walk.Left
	case "right":
		return walk.Right
	default:
		return walk.Forward
	}
}

func defaultMinimapNavigateParam() minimapNavigateParam {
	return minimapNavigateParam{
		TargetNode:    "EchoFarm_BossCheckMinimap",
		MinimapBox:    [4]int{1050, 20, 200, 160},
		ForwardMs:     700,
		Sprint:        true,
		CenterCamera:  true,
		StrongTurnDeg: 135,
		MediumTurnDeg: 45,
		SmallTurnDeg:  12,
	}
}

func normalizeMinimapNavigateParam(param minimapNavigateParam) minimapNavigateParam {
	defaults := defaultMinimapNavigateParam()
	if param.TargetNode == "" {
		param.TargetNode = defaults.TargetNode
	}
	if param.MinimapBox == [4]int{} {
		param.MinimapBox = defaults.MinimapBox
	}
	if param.ForwardMs <= 0 {
		param.ForwardMs = defaults.ForwardMs
	}
	if param.StrongTurnDeg <= 0 {
		param.StrongTurnDeg = defaults.StrongTurnDeg
	}
	if param.MediumTurnDeg <= 0 {
		param.MediumTurnDeg = defaults.MediumTurnDeg
	}
	if param.SmallTurnDeg <= 0 {
		param.SmallTurnDeg = defaults.SmallTurnDeg
	}
	return param
}

func cropMiniMap(img *image.RGBA, box [4]int) *image.RGBA {
	if img == nil {
		return nil
	}
	rect := image.Rect(box[0], box[1], box[0]+box[2], box[1]+box[3]).Intersect(img.Bounds())
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return nil
	}
	return minicv.ImageCropRect(img, rect)
}

func targetAngleFromBox(target maa.Rect, minimapBox [4]int) float64 {
	cx := float64(minimapBox[0]) + float64(minimapBox[2])/2
	cy := float64(minimapBox[1]) + float64(minimapBox[3])/2
	tx := float64(target[0]) + float64(target[2])/2
	ty := float64(target[1]) + float64(target[3])/2
	dx := tx - cx
	dy := ty - cy
	angle := math.Atan2(dx, -dy) * 180 / math.Pi
	if angle < 0 {
		angle += 360
	}
	return angle
}

func angleDeltaDegrees(current float64, target float64) float64 {
	turn := target - current
	for turn > 180 {
		turn -= 360
	}
	for turn < -180 {
		turn += 360
	}
	return turn
}

func navigateMinimapTurn(ctrl *maa.Controller, turn float64, hold time.Duration, sprint bool, param minimapNavigateParam) {
	switch {
	case turn > param.StrongTurnDeg || turn < -param.StrongTurnDeg:
		mouse.RotateCamera(ctrl, 420, 0)
	case turn > param.MediumTurnDeg:
		mouse.RotateCamera(ctrl, 260, 0)
	case turn < -param.MediumTurnDeg:
		mouse.RotateCamera(ctrl, -260, 0)
	case turn > param.SmallTurnDeg:
		pressFor(ctrl, keycode.MustCode("D"), 120*time.Millisecond)
	case turn < -param.SmallTurnDeg:
		pressFor(ctrl, keycode.MustCode("A"), 120*time.Millisecond)
	}

	if sprint {
		walk.Sprint(ctrl, walk.Forward, hold)
	} else {
		walk.Walk(ctrl, walk.Forward, hold)
	}
}

func formatFloat(v float64) string {
	return strings.TrimRight(strings.TrimRight(fmtFloat(v), "0"), ".")
}

func fmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

type BossBookRememberSelectionAction struct{}
type BossBookScrollPageAction struct{}
type BossBookSelectByIndexAction struct{}
type BossBookInputSearchTextAction struct{}
type BossBookTargetVisibleRecognition struct{}
type ForwardApproachUntilAction struct{}
type ForwardApproachResultRecognition struct{}

var _ maa.CustomActionRunner = &BossBookRememberSelectionAction{}
var _ maa.CustomActionRunner = &BossBookScrollPageAction{}
var _ maa.CustomActionRunner = &BossBookSelectByIndexAction{}
var _ maa.CustomActionRunner = &BossBookInputSearchTextAction{}
var _ maa.CustomRecognitionRunner = &BossBookTargetVisibleRecognition{}
var _ maa.CustomActionRunner = &ForwardApproachUntilAction{}
var _ maa.CustomRecognitionRunner = &ForwardApproachResultRecognition{}

var bossBookSelectionState struct {
	sync.Mutex
	param bossTeleportParam
}

var forwardApproachState struct {
	sync.Mutex
	result string
}

type bossTeleportParam struct {
	BossType               string         `json:"boss_type"`
	BossName               string         `json:"boss_name"`
	BossProfile            string         `json:"boss_profile"`
	SerialNumber           int            `json:"serial_number"`
	TotalNumber            int            `json:"total_number"`
	BossLevel              string         `json:"boss_level"`
	WalkAfterTeleport      bool           `json:"walk_after_teleport"`
	WaitWorldAfterTeleport bool           `json:"wait_world_after_teleport"`
	CombatWaitMs           int            `json:"combat_wait_ms"`
	Layout                 bossBookLayout `json:"layout"`
}

type bossBookLayout struct {
	ScrollbarX      int32 `json:"scrollbar_x"`
	ScrollbarTop    int   `json:"scrollbar_top"`
	ScrollbarBottom int   `json:"scrollbar_bottom"`
	ItemX           int32 `json:"item_x"`
	ItemTop         int   `json:"item_top"`
	ItemStep        int   `json:"item_step"`
}

type bossBookScrollPageParam struct {
	Page       int            `json:"page"`
	TotalPages int            `json:"total_pages"`
	Layout     bossBookLayout `json:"layout"`
}

type forwardApproachParam struct {
	Direction              string `json:"direction"`
	TimeoutMs              int    `json:"timeout_ms"`
	StepMs                 int    `json:"step_ms"`
	Sprint                 bool   `json:"sprint"`
	CenterCameraIntervalMs int    `json:"center_camera_interval_ms"`
	StopOnCombat           bool   `json:"stop_on_combat"`
	StopOnFPrompt          bool   `json:"stop_on_f_prompt"`
	StopOnTreasure         bool   `json:"stop_on_treasure"`
	CombatNode             string `json:"combat_node"`
	FPromptNode            string `json:"f_prompt_node"`
	TreasureNode           string `json:"treasure_node"`
}

type forwardApproachResultParam struct {
	Expected string `json:"expected"`
}

func (a *BossBookRememberSelectionAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := parseBossTeleportParam(arg, false)
	saveBossBookSelection(param)
	log.Info().
		Str("component", "BossBookRememberSelection").
		Str("boss_name", param.BossName).
		Str("boss_profile", param.BossProfile).
		Int("serial_number", param.SerialNumber).
		Msg("saved boss book selection")
	return true
}

func (a *BossBookScrollPageAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := bossBookScrollPageParam{Page: 1, TotalPages: 1}
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "BossBookScrollPage").Msg("failed to parse param")
		}
	}
	if param.TotalPages <= 1 || param.Page <= 1 {
		return true
	}

	selection, ok := LoadBossBookSelection()
	if ok && param.Layout == (bossBookLayout{}) {
		param.Layout = selection.Layout
	}
	if param.TotalPages <= 1 && ok {
		if selection.TotalNumber <= 9 {
			param.TotalPages = 3
		} else {
			param.TotalPages = 5
		}
	}
	if param.TotalPages <= 1 || param.Page > param.TotalPages {
		return false
	}
	param.Layout = normalizeBossBookLayout(param.Layout)

	y := param.Layout.ScrollbarTop + (param.Layout.ScrollbarBottom-param.Layout.ScrollbarTop)*(param.Page-1)/(param.TotalPages-1)
	ctx.GetTasker().GetController().PostClick(param.Layout.ScrollbarX, int32(y)).Wait()
	log.Debug().
		Str("component", "BossBookScrollPage").
		Int("page", param.Page).
		Int("total_pages", param.TotalPages).
		Msg("scrolled boss book page")
	return true
}

func (a *BossBookSelectByIndexAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := resolveBossTeleportParam(arg, false)
	selectBookTargetByIndex(ctx, param)
	log.Info().
		Str("component", "BossBookSelectByIndex").
		Int("serial_number", param.SerialNumber).
		Int("total_number", param.TotalNumber).
		Msg("selected boss by index")
	return true
}

func (a *BossBookInputSearchTextAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := resolveBossTeleportParam(arg, false)
	query := preferredBossSearchText(param)
	if query == "" {
		log.Warn().Str("component", "BossBookInputSearchText").Msg("no boss search text available")
		return false
	}

	ctrl := ctx.GetTasker().GetController()
	ctrl.PostKeyDown(keycode.MustCode("CMD")).Wait()
	ctrl.PostClickKey(keycode.MustCode("A")).Wait()
	ctrl.PostKeyUp(keycode.MustCode("CMD")).Wait()
	ctrl.PostInputText(query).Wait()

	log.Info().
		Str("component", "BossBookInputSearchText").
		Str("query", query).
		Msg("input boss search text")
	return true
}

func (r *BossBookTargetVisibleRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := resolveBossTeleportRecognitionParam(arg)
	queries := bossBookQueries(param)
	if len(queries) == 0 {
		return nil, false
	}

	query, box, ok := findBookTargetName(ctx, queries)
	if !ok {
		return nil, false
	}
	if box[1]+box[3]/2 < param.Layout.ItemTop-54 {
		box[1] = param.Layout.ItemTop - box[3]/2
	}

	log.Debug().
		Str("component", "BossBookTargetVisible").
		Str("boss_name", query).
		Str("boss_profile", param.BossProfile).
		Interface("box", box).
		Msg("found boss on current page")
	return &maa.CustomRecognitionResult{
		Box:    box,
		Detail: `{"boss_target_visible":true}`,
	}, true
}

func (a *ForwardApproachUntilAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := defaultForwardApproachParam()
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "ForwardApproachUntil").Msg("failed to parse param")
		}
	}
	param = normalizeForwardApproachParam(param)
	saveForwardApproachResult("timeout")

	ctrl := ctx.GetTasker().GetController()
	dir := parseDirection(param.Direction)
	deadline := time.Now().Add(time.Duration(param.TimeoutMs) * time.Millisecond)
	lastCenter := time.Time{}

	for time.Now().Before(deadline) {
		if ctx.GetTasker().Stopping() {
			saveForwardApproachResult("timeout")
			return false
		}
		if param.StopOnCombat && recognitionHit(ctx, param.CombatNode) {
			saveForwardApproachResult("combat")
			return true
		}
		if param.StopOnFPrompt && recognitionHit(ctx, param.FPromptNode) {
			saveForwardApproachResult("f_prompt")
			return true
		}
		if param.StopOnTreasure && recognitionHit(ctx, param.TreasureNode) {
			saveForwardApproachResult("treasure")
			return true
		}
		if param.CenterCameraIntervalMs > 0 && (lastCenter.IsZero() || time.Since(lastCenter) >= time.Duration(param.CenterCameraIntervalMs)*time.Millisecond) {
			mouse.MiddleClick(ctrl)
			lastCenter = time.Now()
		}

		remaining := time.Until(deadline)
		step := time.Duration(param.StepMs) * time.Millisecond
		if step > remaining {
			step = remaining
		}
		if step <= 0 {
			break
		}
		if param.Sprint {
			walk.Sprint(ctrl, dir, step)
		} else {
			walk.Walk(ctrl, dir, step)
		}
	}

	saveForwardApproachResult("timeout")
	log.Info().
		Str("component", "ForwardApproachUntil").
		Str("result", "timeout").
		Int("timeout_ms", param.TimeoutMs).
		Msg("approach timed out")
	return true
}

func (r *ForwardApproachResultRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := forwardApproachResultParam{}
	if arg != nil && arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "ForwardApproachResult").Msg("failed to parse param")
		}
	}
	if strings.TrimSpace(param.Expected) == "" {
		return nil, false
	}

	result := loadForwardApproachResult()
	if result != param.Expected {
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"forward_approach_result":"` + result + `"}`,
	}, true
}

func saveBossBookSelection(param bossTeleportParam) {
	bossBookSelectionState.Lock()
	bossBookSelectionState.param = param
	bossBookSelectionState.Unlock()
}

func LoadBossBookSelection() (bossTeleportParam, bool) {
	bossBookSelectionState.Lock()
	defer bossBookSelectionState.Unlock()
	if bossBookSelectionState.param.BossLevel == "" && bossBookSelectionState.param.BossProfile == "" {
		return bossTeleportParam{}, false
	}
	return bossBookSelectionState.param, true
}

func resolveBossTeleportParam(arg *maa.CustomActionArg, waitWorldDefault bool) bossTeleportParam {
	if arg != nil && arg.CustomActionParam != "" {
		return parseBossTeleportParam(arg, waitWorldDefault)
	}
	if saved, ok := LoadBossBookSelection(); ok {
		saved.Layout = normalizeBossBookLayout(saved.Layout)
		return saved
	}
	return parseBossTeleportParam(arg, waitWorldDefault)
}

func resolveBossTeleportRecognitionParam(arg *maa.CustomRecognitionArg) bossTeleportParam {
	param := bossTeleportParam{
		BossType:     "boss",
		SerialNumber: 1,
		TotalNumber:  20,
		BossLevel:    "80",
	}
	if saved, ok := LoadBossBookSelection(); ok {
		param = saved
	}
	if arg != nil && arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "BossBookTargetVisible").Msg("failed to parse param")
		}
	}
	if param.SerialNumber < 1 {
		param.SerialNumber = 1
	}
	if param.TotalNumber < param.SerialNumber {
		param.TotalNumber = param.SerialNumber
	}
	if param.BossType == "weekly" && param.TotalNumber < 9 {
		param.TotalNumber = 9
	}
	if param.BossLevel == "" {
		param.BossLevel = "80"
	}
	param.Layout = normalizeBossBookLayout(param.Layout)
	return param
}

func parseBossTeleportParam(arg *maa.CustomActionArg, waitWorldDefault bool) bossTeleportParam {
	param := bossTeleportParam{
		BossType:               "boss",
		SerialNumber:           1,
		TotalNumber:            20,
		BossLevel:              "80",
		WalkAfterTeleport:      true,
		WaitWorldAfterTeleport: waitWorldDefault,
	}
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "BossBookParam").Msg("failed to parse param")
		}
	}
	if param.SerialNumber < 1 {
		param.SerialNumber = 1
	}
	if param.TotalNumber < param.SerialNumber {
		param.TotalNumber = param.SerialNumber
	}
	if param.BossType == "weekly" && param.TotalNumber < 9 {
		param.TotalNumber = 9
	}
	if param.BossLevel == "" {
		param.BossLevel = "80"
	}
	param.Layout = normalizeBossBookLayout(param.Layout)
	return param
}

func normalizeBossBookLayout(layout bossBookLayout) bossBookLayout {
	if layout.ScrollbarX == 0 {
		layout.ScrollbarX = 1245
	}
	if layout.ScrollbarTop == 0 {
		layout.ScrollbarTop = 112
	}
	if layout.ScrollbarBottom == 0 {
		layout.ScrollbarBottom = 634
	}
	if layout.ItemX == 0 {
		layout.ItemX = 1195
	}
	if layout.ItemTop == 0 {
		layout.ItemTop = 174
	}
	if layout.ItemStep == 0 {
		layout.ItemStep = 118
	}
	return layout
}

func selectBookTargetByIndex(ctx *maa.Context, param bossTeleportParam) {
	ctrl := ctx.GetTasker().GetController()
	row := param.SerialNumber
	if row > 4 {
		y := param.Layout.ScrollbarTop + (param.Layout.ScrollbarBottom-param.Layout.ScrollbarTop)*param.SerialNumber/param.TotalNumber
		ctrl.PostClick(param.Layout.ScrollbarX, int32(y)).Wait()
		row = 4
	}
	y := int32(param.Layout.ItemTop + (row-1)*param.Layout.ItemStep)
	ctrl.PostClick(param.Layout.ItemX, y).Wait()
}

func findBookTargetName(ctx *maa.Context, queries []string) (string, maa.Rect, bool) {
	detail, err := ctx.RunRecognition("Navigation_BossNameOCR", nil)
	if err != nil || detail == nil || !detail.Hit || detail.Results == nil {
		return "", maa.Rect{}, false
	}

	results := detail.Results.Filtered
	if len(results) == 0 {
		results = detail.Results.All
	}
	for _, result := range results {
		ocr, ok := result.AsOCR()
		if !ok || ocr == nil {
			continue
		}
		text := normalizeBossBookText(ocr.Text)
		for _, query := range queries {
			if query != "" && strings.Contains(text, query) {
				return query, ocr.Box, true
			}
		}
	}
	return "", maa.Rect{}, false
}

func normalizeBossBookText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, " ", "")
	text = strings.ReplaceAll(text, "\n", "")
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ReplaceAll(text, "：", ":")
	return strings.ToLower(text)
}

func bossBookQueries(param bossTeleportParam) []string {
	seen := map[string]struct{}{}
	queries := make([]string, 0, 6)
	appendQuery := func(raw string) {
		q := normalizeBossBookText(raw)
		if q == "" {
			return
		}
		if _, ok := seen[q]; ok {
			return
		}
		seen[q] = struct{}{}
		queries = append(queries, q)
	}

	appendQuery(param.BossName)
	for _, alias := range bossProfileAliases(param.BossProfile) {
		appendQuery(alias)
	}
	return queries
}

func preferredBossSearchText(param bossTeleportParam) string {
	if name := strings.TrimSpace(param.BossName); name != "" {
		return name
	}
	switch normalizeBossBookText(param.BossProfile) {
	case "hyvatia":
		return "海维夏"
	case "fallacy", "fallacyofnoreturn":
		return "无归的谬误"
	case "sentryconstruct":
		return "异构武装"
	case "lorelei":
		return "罗蕾莱"
	case "lionessofglory":
		return "荣耀狮像"
	case "nightmarehecate":
		return "梦魇赫卡忒"
	case "fenrico":
		return "芬莱克"
	case "namelessexplorer":
		return "无铭探索者"
	case "ladyofthesea":
		return "海之女"
	default:
		return ""
	}
}

func bossProfileAliases(profile string) []string {
	switch normalizeBossBookText(profile) {
	case "hyvatia":
		return []string{"hyvatia", "海维夏"}
	case "fallacy", "fallacyofnoreturn":
		return []string{"fallacyofnoreturn", "无归的谬误"}
	case "sentryconstruct":
		return []string{"sentryconstruct", "异构武装", "加尔古耶"}
	case "lorelei":
		return []string{"lorelei", "罗蕾莱", "夜之女皇"}
	case "lionessofglory":
		return []string{"lionessofglory", "荣耀狮像", "亚狮诺索"}
	case "nightmarehecate":
		return []string{"nightmare:hecate", "nightmarehecate", "梦魇赫卡忒", "梦魇:赫卡忒", "梦魇·赫卡忒"}
	case "fenrico":
		return []string{"fenrico", "芬莱克"}
	case "namelessexplorer":
		return []string{"namelessexplorer", "无铭探索者"}
	case "ladyofthesea":
		return []string{"ladyofthesea", "lady_of_the_sea", "lady of the sea", "海之女"}
	default:
		return nil
	}
}

func defaultForwardApproachParam() forwardApproachParam {
	return forwardApproachParam{
		Direction:              "forward",
		TimeoutMs:              12000,
		StepMs:                 500,
		Sprint:                 true,
		CenterCameraIntervalMs: 0,
		StopOnCombat:           true,
		StopOnFPrompt:          true,
		StopOnTreasure:         true,
		CombatNode:             "EchoFarm_TeleportWalkCombat",
		FPromptNode:            "EchoFarm_TeleportWalkFPrompt",
		TreasureNode:           "EchoFarm_TeleportWalkTreasure",
	}
}

func normalizeForwardApproachParam(param forwardApproachParam) forwardApproachParam {
	defaults := defaultForwardApproachParam()
	if param.Direction == "" {
		param.Direction = defaults.Direction
	}
	if param.TimeoutMs <= 0 {
		param.TimeoutMs = defaults.TimeoutMs
	}
	if param.StepMs <= 0 {
		param.StepMs = defaults.StepMs
	}
	if param.CombatNode == "" {
		param.CombatNode = defaults.CombatNode
	}
	if param.FPromptNode == "" {
		param.FPromptNode = defaults.FPromptNode
	}
	if param.TreasureNode == "" {
		param.TreasureNode = defaults.TreasureNode
	}
	return param
}

func saveForwardApproachResult(result string) {
	forwardApproachState.Lock()
	forwardApproachState.result = result
	forwardApproachState.Unlock()
}

func loadForwardApproachResult() string {
	forwardApproachState.Lock()
	defer forwardApproachState.Unlock()
	return forwardApproachState.result
}

func recognitionHit(ctx *maa.Context, node string) bool {
	if strings.TrimSpace(node) == "" {
		return false
	}
	detail, err := ctx.RunRecognition(node, nil)
	return err == nil && detail != nil && detail.Hit
}

func pressFor(ctrl *maa.Controller, code int32, duration time.Duration) {
	ctrl.PostKeyDown(code).Wait()
	time.Sleep(duration)
	ctrl.PostKeyUp(code).Wait()
}

func clickRecognitionCenter(ctx *maa.Context, node string) bool {
	detail, err := ctx.RunRecognition(node, nil)
	if err != nil || detail == nil || !detail.Hit {
		return false
	}
	ctx.GetTasker().GetController().PostClick(
		int32(detail.Box[0]+detail.Box[2]/2),
		int32(detail.Box[1]+detail.Box[3]/2),
	).Wait()
	return true
}

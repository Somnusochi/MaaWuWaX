package combat

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/minicv"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/resource"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

// ScreenAnalyzer — ok-ww CombatCheck port: one frame, batch detections.
type ScreenAnalyzer struct {
	HasTarget        bool
	HasDodge         bool
	HasHPBar         bool
	HasBossHP        bool
	PickupF          bool
	Liberation       bool
	CharAlive        [3]bool
	CharSlots        [3]charSlot
	CurrentIdx       int
	TeamSize         int
	ConcertoPct      float64
	RingElement      int // -1 = unknown
	ResonancePct     float64
	EchoPct          float64
	LiberationPct    float64
	ResonanceCD      bool
	EchoCD           bool
	LiberationCD     bool
	FortePct         float64
	Flying           bool
	ForteFull        bool
	MouseForteFull   bool
	EForteFull       bool
	HasLongAction    bool
	HasLongAction2   bool
	LinnaiColorPct   float64
	HiyukiLibForte   bool
	HiyukiLeft       bool
	HiyukiRight      bool
	AemeathEnhanceE  bool
	AemeathLib2      bool
	LupaWolfReady    bool
	CartethyiaSword1 bool
	CartethyiaSword2 bool
	CartethyiaSword3 bool
	CartethyiaBigLib bool
	CartethyiaSmall  bool
	CartethyiaMidAir bool
	LuhesiKickReady  bool
	LuhesiLibReady   bool
	IunoHeavyReady   bool
	IunoJumpReady    bool
	AugustaLibReady  bool
	AugustaMajesty   bool
	AugustaProwess   bool
	LinnaiCheckRes   bool
	LuhesiCheckRes   bool
	GalbrenaCheckRes bool
	XigelikaForte    bool
	CamellyaBudding  bool
	ChangliFortePct  float64
	ZhezhiForteTier  int
	CiacconaForte    int
	LupaForte        int
	PhoebeLightForte int
	PhoebeBlueForte  int
	PhoebeFullLight  bool
	PhoebeFullBlue   bool
	CarlottaForte    int
	CamellyaFortePct float64
	CamellyaBudPct   float64
	ZaniFortePct     float64
	ZhezhiBluePct    float64
	PhoebeStarLight  float64
	PhoebeStarBlue   float64
	PhoebeRingBlue   float64
	CamellyaRedPct   float64
	ZaniNightfallPct float64
	ZaniBlazesPct    float64
	ZaniLiberBox     bool
	ZaniNotLiberBox  bool

	// Freeze/time-stop compensation (ok-ww: time_elapsed_accounting_for_freeze)
	LastFrameTime   int64 // unix nano
	FreezeDuration  int64 // accumulated freeze in nanoseconds
	FreezeThreshold int64 // 100ms default (ok-ww default: 0.1s)

	// Health-based switching
	HealthLow [3]bool // per-slot low-health flag
}

var screenAnalyzer = &ScreenAnalyzer{
	RingElement:     -1,
	CurrentIdx:      -1,
	FreezeThreshold: 100_000_000, // 100ms
}

const (
	ringElementUnknown = -1
	ringElementSpectro = 0
	ringElementWind    = 4
	ringElementHavoc   = 5
)

var combatCDPattern = regexp.MustCompile(`\d{1,2}\.\d`)

var charTemplateLoaders sync.Map // map[string]*minicv.TemplateLoader

// Update runs all recognition nodes against one frame.
func (sa *ScreenAnalyzer) Update(ctx *maa.Context, img image.Image) bool {
	if img == nil {
		return false
	}

	// 1. Target lock (ok-ww: has_target, threshold 0.6)
	detail, err := ctx.RunRecognition("Combat_AnalyzerTarget", img)
	sa.HasTarget = err == nil && detail != nil && detail.Hit

	// 2. Dodge prompt (ok-ww: dodge_prompt, threshold 0.6)
	detail, err = ctx.RunRecognition("Combat_AnalyzerDodge", img)
	sa.HasDodge = err == nil && detail != nil && detail.Hit

	// 3. Red HP bar via ColorMatch (ok-ww: enemy_health_color_red)
	detail, err = ctx.RunRecognition("Combat_AnalyzerHP", img)
	sa.HasHPBar = err == nil && detail != nil && detail.Hit

	// 4. Boss HP bar
	detail, err = ctx.RunRecognition("Combat_AnalyzerBossHP", img)
	sa.HasBossHP = err == nil && detail != nil && detail.Hit

	// 5. F pickup prompt
	detail, err = ctx.RunRecognition("Combat_AnalyzerPick", img)
	sa.PickupF = err == nil && detail != nil && detail.Hit

	// 6. Liberation availability hint.
	detail, err = ctx.RunRecognition("Combat_AnalyzerLiberationReady", img)
	sa.Liberation = err == nil && detail != nil && detail.Hit

	sa.RingElement = ringElementUnknown
	ringChecks := []struct {
		name    string
		element int
	}{
		{"Combat_RingConFullHavoc", ringElementHavoc},
		{"Combat_RingConFullSpectro", ringElementSpectro},
		{"Combat_RingConFullWind", ringElementWind},
		{"Combat_RingConHavoc", ringElementHavoc},
		{"Combat_RingConSpectro", ringElementSpectro},
		{"Combat_RingConWind", ringElementWind},
		{"Combat_RingLibReadyHavoc", ringElementHavoc},
		{"Combat_RingLibReadySpectro", ringElementSpectro},
		{"Combat_RingLibReadyWind", ringElementWind},
	}
	for _, rc := range ringChecks {
		detail, err = ctx.RunRecognition(rc.name, img)
		if err == nil && detail != nil && detail.Hit {
			sa.RingElement = rc.element
			break
		}
	}

	// 6.5. Shared UI percentages/state used by many ok-ww character strategies.
	sa.ResonancePct = sampleNearWhitePct(img, maa.Rect{1062, 626, 45, 38})
	sa.EchoPct = sampleNearWhitePct(img, maa.Rect{1124, 626, 45, 39})
	sa.LiberationPct = sampleNearWhitePct(img, maa.Rect{1190, 626, 42, 38})
	sa.refreshCooldowns(ctx, img)
	sa.FortePct = sampleNearWhitePct(img, maa.Rect{750, 664, 20, 8})
	sa.Flying = sampleNearWhitePct(img, maa.Rect{1005, 629, 27, 27}) < 0.1
	sa.ZhezhiBluePct = sampleColorPct(img, maa.Rect{560, 666, 15, 7}, 160, 180, 240, 255, 245, 255)
	sa.PhoebeStarLight = sampleColorPct(img, maa.Rect{630, 670, 8, 7}, 240, 255, 240, 255, 180, 220)
	sa.PhoebeStarBlue = sampleColorPct(img, maa.Rect{630, 670, 8, 7}, 140, 200, 210, 255, 240, 255)
	sa.PhoebeRingBlue = sampleColorPct(img, maa.Rect{1055, 618, 54, 54}, 140, 200, 210, 255, 240, 255)
	sa.CamellyaRedPct = sampleColorPct(img, maa.Rect{1055, 618, 54, 54}, 180, 255, 20, 120, 20, 120)
	sa.ChangliFortePct = sampleStripeFillPct(img, maa.Rect{544, 668, 176, 4}, 240, 255, 85, 105, 95, 115)
	sa.ZhezhiForteTier = sampleForteNumByFFT(img, scaledRect(5120, 2880, 2164, 2675, 2900, 2685), 736.0/3.0, colorRange{185, 215, 240, 255, 235, 255}, 3, 12, 14, 100, true)
	sa.CiacconaForte = sampleForteNumByFFT(img, scaledRect(3840, 2160, 1612, 1987, 2188, 2008), 576.0/3.0, colorRange{70, 100, 240, 255, 180, 210}, 3, 12, 14, 100, true)
	sa.LupaForte = sampleForteNumByFFT(img, scaledRect(3840, 2160, 1633, 2004, 2160, 2016), 527.0/2.0, colorRange{235, 255, 75, 105, 75, 105}, 2, 19, 21, 400, true)
	phoebeForteBox := scaledRect(3840, 2160, 1633, 2004, 2160, 2014)
	sa.PhoebeLightForte = sampleForteNumByFFT(img, phoebeForteBox, 527.0/4.0, colorRange{240, 255, 240, 255, 165, 195}, 4, 9, 11, 25, false)
	sa.PhoebeBlueForte = sampleForteNumByFFT(img, phoebeForteBox, 527.0/2.0, colorRange{225, 255, 225, 255, 190, 225}, 2, 18, 20, 50, false)
	sa.PhoebeFullLight = sampleForteFullByWhiteContrast(img, scaledRect(3840, 2160, 2286, 1992, 2306, 2018), 0.08)
	sa.PhoebeFullBlue = sampleForteFullByWhiteContrast(img, scaledRect(3840, 2160, 2256, 1992, 2276, 2018), 0.08)
	sa.CarlottaForte = sampleForteNumByFFT(img, scaledRect(5120, 2880, 2164, 2670, 2900, 2680), 736.0/4.0, colorRange{70, 100, 195, 225, 235, 255}, 4, 9, 11, 100, true)
	sa.XigelikaForte = sampleNearWhitePct(img, scaledRect(5120, 2880, 3032, 2654, 3076, 2700)) > 0.1
	sa.CamellyaFortePct = sampleStripeFillPct(img, maa.Rect{543, 667, 182, 2}, 193, 255, 46, 93, 127, 163)
	sa.CamellyaBudPct = sampleStripeFillPct(img, maa.Rect{543, 667, 182, 2}, 220, 255, 161, 213, 168, 225)
	sa.ZaniFortePct = sampleColorPct(img, maa.Rect{543, 665, 185, 3}, 239, 255, 222, 255, 156, 196)
	sa.ZaniNightfallPct = sampleColorPct(img, maa.Rect{926, 616, 55, 56}, 210, 255, 215, 255, 180, 255)
	sa.ZaniBlazesPct = sampleColorPct(img, maa.Rect{543, 671, 183, 3}, 240, 255, 210, 255, 150, 220)
	sa.LinnaiColorPct = sampleNearWhitePct(img, maa.Rect{711, 650, 11, 22})

	detail, err = ctx.RunRecognition("Combat_ZaniNotLiber", img)
	sa.ZaniNotLiberBox = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_ZaniLiber", img)
	sa.ZaniLiberBox = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_MouseForte", img)
	sa.MouseForteFull = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_EForte", img)
	sa.EForteFull = err == nil && detail != nil && detail.Hit
	sa.ForteFull = sa.FortePct > 0.08 || sa.EForteFull

	detail, err = ctx.RunRecognition("Combat_LongAction1", img)
	sa.HasLongAction = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_LongAction2", img)
	sa.HasLongAction2 = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_HiyukiLibForte", img)
	sa.HiyukiLibForte = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_HiyukiLeft", img)
	sa.HiyukiLeft = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_HiyukiRight", img)
	sa.HiyukiRight = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_AemeathEnhanceE", img)
	enhance1 := err == nil && detail != nil && detail.Hit
	detail, err = ctx.RunRecognition("Combat_AemeathEnhanceE2", img)
	enhance2 := err == nil && detail != nil && detail.Hit
	sa.AemeathEnhanceE = enhance1 || enhance2

	detail, err = ctx.RunRecognition("Combat_AemeathLib2", img)
	sa.AemeathLib2 = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_LupaWolfReady", img)
	sa.LupaWolfReady = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_CartethyiaSword1", img)
	sa.CartethyiaSword1 = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_CartethyiaSword2", img)
	sa.CartethyiaSword2 = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_CartethyiaSword3", img)
	sa.CartethyiaSword3 = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_CartethyiaBigLib", img)
	sa.CartethyiaBigLib = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_CartethyiaSmall", img)
	sa.CartethyiaSmall = err == nil && detail != nil && detail.Hit
	if sa.CartethyiaSword3 {
		sa.CartethyiaSmall = true
	}

	cartethyiaMidAirBox := maa.Rect{766, 665, 21, 9}
	if sampleNearWhitePct(img, cartethyiaMidAirBox) > 0.15 {
		mean, std := sampleGrayMeanStd(img, cartethyiaMidAirBox)
		sa.CartethyiaMidAir = mean > 190 && std < 45
	} else {
		sa.CartethyiaMidAir = false
	}

	detail, err = ctx.RunRecognition("Combat_LuhesiKickReady", img)
	sa.LuhesiKickReady = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_LuhesiLibReady", img)
	sa.LuhesiLibReady = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_IunoHeavyReady", img)
	sa.IunoHeavyReady = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_IunoJumpReady", img)
	sa.IunoJumpReady = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_AugustaLibReady", img)
	sa.AugustaLibReady = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_AugustaMajesty", img)
	sa.AugustaMajesty = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("Combat_AugustaProwess", img)
	sa.AugustaProwess = err == nil && detail != nil && detail.Hit
	sa.LinnaiCheckRes = sa.anyPipelineHit(ctx, img, "Combat_LinnaiCheckResTarget", "Combat_LinnaiCheckResNoTarget")
	sa.LuhesiCheckRes = sa.anyPipelineHit(ctx, img, "Combat_LuhesiCheckResTarget", "Combat_LuhesiCheckResNoTarget")
	sa.GalbrenaCheckRes = sa.anyPipelineHit(ctx, img, "Combat_GalbrenaCheckResTarget", "Combat_GalbrenaCheckResNoTarget")

	detail, err = ctx.RunRecognition("Combat_CamellyaBudding", img)
	sa.CamellyaBudding = err == nil && detail != nil && detail.Hit

	// 7. Character portraits
	sa.ConcertoPct = 0
	existCount := 0
	currentIdx := -1
	for i := range 3 {
		detail, err = ctx.RunRecognition(fmt.Sprintf("Combat_AnalyzerChar%d", i+1), img)
		textHit := err == nil && detail != nil && detail.Hit
		if textHit {
			existCount++
		} else if currentIdx < 0 {
			currentIdx = i
		}

		detail, err = ctx.RunRecognition(fmt.Sprintf("Combat_AnalyzerCon%d", i+1), img)
		if err == nil && detail != nil && detail.Hit {
			sa.ConcertoPct = 1.0
		}
	}
	if existCount == 1 || existCount == 2 {
		sa.CurrentIdx = currentIdx
		sa.TeamSize = existCount + 1
	} else {
		sa.CurrentIdx = -1
		sa.TeamSize = 0
	}
	for i := range 3 {
		sa.CharAlive[i] = sa.TeamSize > i
		slot := sa.detectCharSlot(ctx, img, i)
		slot.Index = i
		slot.Alive = sa.CharAlive[i]
		slot.Current = sa.CurrentIdx == i
		sa.CharSlots[i] = slot
	}

	log.Debug().Str("component", "ScreenAnalyzer").
		Bool("target", sa.HasTarget).Bool("hp", sa.HasHPBar).
		Bool("boss", sa.HasBossHP).Bool("dodge", sa.HasDodge).Bool("liberation", sa.Liberation).
		Bool("flying", sa.Flying).Bool("forte_full", sa.ForteFull).
		Int("current", sa.CurrentIdx+1).Int("team_size", sa.TeamSize).
		Int("ring", sa.RingElement).
		Float64("concerto", sa.ConcertoPct).
		Float64("resonance_pct", sa.ResonancePct).
		Float64("forte_pct", sa.FortePct).
		Bool("aemeath_e", sa.AemeathEnhanceE).
		Bool("aemeath_lib2", sa.AemeathLib2).
		Bool("lupa_wolf", sa.LupaWolfReady).
		Bool("cartethyia_sword1", sa.CartethyiaSword1).
		Bool("cartethyia_sword2", sa.CartethyiaSword2).
		Bool("cartethyia_sword3", sa.CartethyiaSword3).
		Bool("cartethyia_biglib", sa.CartethyiaBigLib).
		Bool("cartethyia_small", sa.CartethyiaSmall).
		Bool("cartethyia_midair", sa.CartethyiaMidAir).
		Bool("luhesi_kick", sa.LuhesiKickReady).
		Bool("luhesi_lib", sa.LuhesiLibReady).
		Bool("iuno_heavy", sa.IunoHeavyReady).
		Bool("iuno_jump", sa.IunoJumpReady).
		Float64("linnai_color", sa.LinnaiColorPct).
		Bool("linnai_check_res", sa.LinnaiCheckRes).
		Bool("luhesi_check_res", sa.LuhesiCheckRes).
		Bool("galbrena_check_res", sa.GalbrenaCheckRes).
		Bool("xigelika_forte", sa.XigelikaForte).
		Float64("zhezhi_blue", sa.ZhezhiBluePct).
		Int("zhezhi_forte", sa.ZhezhiForteTier).
		Int("ciaccona_forte", sa.CiacconaForte).
		Int("lupa_forte", sa.LupaForte).
		Int("phoebe_light_forte", sa.PhoebeLightForte).
		Int("phoebe_blue_forte", sa.PhoebeBlueForte).
		Int("carlotta_forte", sa.CarlottaForte).
		Float64("changli_forte", sa.ChangliFortePct).
		Float64("phoebe_star_light", sa.PhoebeStarLight).
		Float64("camellya_red", sa.CamellyaRedPct).
		Float64("camellya_forte", sa.CamellyaFortePct).
		Float64("camellya_bud", sa.CamellyaBudPct).
		Float64("zani_forte", sa.ZaniFortePct).
		Float64("zani_nightfall", sa.ZaniNightfallPct).
		Float64("zani_blazes", sa.ZaniBlazesPct).
		Bool("zani_liber_box", sa.ZaniLiberBox).
		Bool("zani_not_liber_box", sa.ZaniNotLiberBox).Msg("frame")

	return sa.InCombat()
}

// InCombat — ok-ww: has_target() OR check_health_bar()
func (sa *ScreenAnalyzer) InCombat() bool {
	return sa.HasTarget || sa.HasHPBar || sa.HasBossHP
}

func (sa *ScreenAnalyzer) detectCharSlot(ctx *maa.Context, img image.Image, index int) charSlot {
	slot := charSlot{Index: index, Role: roleUnknown}
	box, ok := sa.findCharBox(ctx, img, index)
	if !ok {
		return slot
	}
	imgRGBA := minicv.ImageConvertRGBA(img)
	imgIntegral := minicv.GetIntegralArray(imgRGBA)
	bestScore := 0.0
	bestMeta := charMeta{}
	for _, meta := range charTemplates {
		score := matchCharTemplateInBox(imgRGBA, imgIntegral, meta.Template, box)
		if score < 0.55 {
			continue
		}
		if score > bestScore {
			bestScore = score
			bestMeta = meta
		}
	}
	if bestScore <= 0 {
		return slot
	}
	slot.Name = bestMeta.Name
	slot.Role = bestMeta.Role
	slot.Detected = true
	return slot
}

func matchCharTemplateInBox(img *image.RGBA, imgIntegral minicv.IntegralArray, template string, box maa.Rect) float64 {
	tpl, err := charTemplateLoader(template).Get()
	if err != nil {
		log.Debug().Err(err).Str("template", template).Msg("char template unavailable")
		return 0
	}
	_, _, score := minicv.MatchTemplateInArea(
		img,
		imgIntegral,
		tpl.Image,
		tpl.Stats,
		[4]int{box[0], box[1], box[2], box[3]},
	)
	return score
}

func charTemplateLoader(template string) *minicv.TemplateLoader {
	if loader, ok := charTemplateLoaders.Load(template); ok {
		return loader.(*minicv.TemplateLoader)
	}
	loader := minicv.NewTemplateLoaderOfDynamicPath(func() string {
		return resolveResourceImagePath(template)
	})
	actual, _ := charTemplateLoaders.LoadOrStore(template, loader)
	return actual.(*minicv.TemplateLoader)
}

func resolveResourceImagePath(template string) string {
	bases := make([]string, 0, len(resource.GetStandardResourceBase())+1)
	if base := resource.GetResourceBase(); base != "" {
		bases = append(bases, base)
	}
	bases = append(bases, resource.GetStandardResourceBase()...)

	for _, base := range bases {
		for _, candidate := range []string{
			filepath.Join(base, "image", template),
			filepath.Join(base, "resource", "image", template),
			filepath.Join(base, "assets", "resource", "image", template),
		} {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	return ""
}

// TrackFrame records the current frame timestamp and detects freezes (>100ms gap).
// Implements ok-ww time_elapsed_accounting_for_freeze for CD compensation.
func (sa *ScreenAnalyzer) TrackFrame(now int64) {
	if sa.LastFrameTime > 0 {
		gap := now - sa.LastFrameTime
		if gap > sa.FreezeThreshold {
			sa.FreezeDuration += gap - sa.FreezeThreshold
		}
	}
	sa.LastFrameTime = now
}

// ElapsedSince returns wall-clock duration adjusted for accumulated freeze time.
func (sa *ScreenAnalyzer) ElapsedSince(sinceUnixNano int64) time.Duration {
	elapsed := time.Now().UnixNano() - sinceUnixNano - sa.FreezeDuration
	if elapsed < 0 {
		elapsed = 0
	}
	return time.Duration(elapsed)
}

// HasHealthLow checks if the current character's health bar indicates low HP.
func (sa *ScreenAnalyzer) HasHealthLow() bool {
	if sa.CurrentIdx >= 0 && sa.CurrentIdx < 3 {
		return sa.HealthLow[sa.CurrentIdx]
	}
	return false
}

func (sa *ScreenAnalyzer) findCharBox(ctx *maa.Context, img image.Image, index int) (maa.Rect, bool) {
	detail, err := ctx.RunRecognition(fmt.Sprintf("Combat_BoxChar%d", index+1), img)
	if err == nil && detail != nil && detail.Hit {
		box := detail.Box
		return maa.Rect{
			int(math.Max(0, float64(box[0]-12))),
			int(math.Max(0, float64(box[1]-12))),
			box[2] + 24,
			box[3] + 24,
		}, true
	}
	fallback := []maa.Rect{
		{1070, 140, 190, 170},
		{1070, 285, 190, 170},
		{1070, 430, 190, 170},
	}[index]
	return fallback, true
}

func (sa *ScreenAnalyzer) anyPipelineHit(ctx *maa.Context, img image.Image, nodes ...string) bool {
	for _, nodeName := range nodes {
		detail, err := ctx.RunRecognition(nodeName, img)
		if err == nil && detail != nil && detail.Hit {
			return true
		}
	}
	return false
}

type colorRange struct {
	rMin, rMax uint32
	gMin, gMax uint32
	bMin, bMax uint32
}

func scaledRect(srcW, srcH, x1, y1, x2, y2 int) maa.Rect {
	left := int(math.Round(float64(x1) * 1280 / float64(srcW)))
	top := int(math.Round(float64(y1) * 720 / float64(srcH)))
	right := int(math.Round(float64(x2) * 1280 / float64(srcW)))
	bottom := int(math.Round(float64(y2) * 720 / float64(srcH)))
	return maa.Rect{left, top, maxInt(1, right-left), maxInt(1, bottom-top)}
}

func sampleNearWhitePct(img image.Image, box maa.Rect) float64 {
	if img == nil || box[2] <= 0 || box[3] <= 0 {
		return 0
	}
	bounds := img.Bounds()
	x1 := maxInt(bounds.Min.X, box[0])
	y1 := maxInt(bounds.Min.Y, box[1])
	x2 := minInt(bounds.Max.X, box[0]+box[2])
	y2 := minInt(bounds.Max.Y, box[1]+box[3])
	if x2 <= x1 || y2 <= y1 {
		return 0
	}
	total := 0
	white := 0
	for y := y1; y < y2; y++ {
		for x := x1; x < x2; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r>>8 >= 244 && g>>8 >= 246 && b>>8 >= 250 {
				white++
			}
			total++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(white) / float64(total)
}

func sampleColorPct(img image.Image, box maa.Rect, rMin, rMax, gMin, gMax, bMin, bMax uint32) float64 {
	if img == nil || box[2] <= 0 || box[3] <= 0 {
		return 0
	}
	bounds := img.Bounds()
	x1 := maxInt(bounds.Min.X, box[0])
	y1 := maxInt(bounds.Min.Y, box[1])
	x2 := minInt(bounds.Max.X, box[0]+box[2])
	y2 := minInt(bounds.Max.Y, box[1]+box[3])
	if x2 <= x1 || y2 <= y1 {
		return 0
	}
	total := 0
	hit := 0
	for y := y1; y < y2; y++ {
		for x := x1; x < x2; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8, g8, b8 := r>>8, g>>8, b>>8
			if r8 >= rMin && r8 <= rMax && g8 >= gMin && g8 <= gMax && b8 >= bMin && b8 <= bMax {
				hit++
			}
			total++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(hit) / float64(total)
}

func sampleForteFullByWhiteContrast(img image.Image, box maa.Rect, whiteThreshold float64) bool {
	if sampleNearWhitePct(img, box) <= whiteThreshold {
		return false
	}
	mean, std := sampleGrayMeanStd(img, box)
	return mean > 190 && std < 50
}

func sampleForteNumByFFT(img image.Image, box maa.Rect, sourceStepWidth float64, cr colorRange, num, minFreq, maxFreq int, minAmp float64, scanFromRight bool) int {
	if img == nil || box[2] <= 0 || box[3] <= 0 || num <= 0 {
		return 0
	}
	mask, width, height := colorMask(img, box, cr)
	if width == 0 || height == 0 {
		return 0
	}
	step := width / num
	if step <= 0 {
		return 0
	}
	if scanFromRight {
		for forte := num; forte > 0; forte-- {
			left := step * (forte - 1)
			right := left + step
			if forte == num {
				right = width
			}
			if judgeForteSegment(mask, left, right, sourceStepWidth, minFreq, maxFreq, minAmp) {
				return forte
			}
		}
		return 0
	}

	forte := 0
	failCount := 0
	for left := 0; left+step < width; left += step {
		right := left + step
		if judgeForteSegment(mask, left, right, sourceStepWidth, minFreq, maxFreq, minAmp) {
			if failCount == 0 {
				forte++
			}
		} else {
			failCount++
		}
	}
	return forte
}

func colorMask(img image.Image, box maa.Rect, cr colorRange) ([][]bool, int, int) {
	bounds := img.Bounds()
	x1 := maxInt(bounds.Min.X, box[0])
	y1 := maxInt(bounds.Min.Y, box[1])
	x2 := minInt(bounds.Max.X, box[0]+box[2])
	y2 := minInt(bounds.Max.Y, box[1]+box[3])
	if x2 <= x1 || y2 <= y1 {
		return nil, 0, 0
	}
	width := x2 - x1
	height := y2 - y1
	mask := make([][]bool, height)
	for y := 0; y < height; y++ {
		mask[y] = make([]bool, width)
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x1+x, y1+y).RGBA()
			r8, g8, b8 := r>>8, g>>8, b>>8
			mask[y][x] = r8 >= cr.rMin && r8 <= cr.rMax && g8 >= cr.gMin && g8 <= cr.gMax && b8 >= cr.bMin && b8 <= cr.bMax
		}
	}
	return mask, width, height
}

func judgeForteSegment(mask [][]bool, left, right int, sourceStepWidth float64, minFreq, maxFreq int, minAmp float64) bool {
	height := len(mask)
	if height == 0 || right <= left {
		return false
	}
	width := len(mask[0])
	left = maxInt(0, left)
	right = minInt(width, right)
	segWidth := right - left
	if segWidth <= 0 {
		return false
	}
	profile := make([]float64, segWidth)
	white := 0
	for x := 0; x < segWidth; x++ {
		count := 0
		for y := 0; y < height; y++ {
			if mask[y][left+x] {
				count++
				white++
			}
		}
		profile[x] = float64(count)
	}
	if white == 0 {
		return false
	}

	peakFreq, peakAmp := fftPeakWithFrequency(profile)
	if sourceStepWidth <= 0 {
		sourceStepWidth = float64(segWidth)
	}
	scale := float64(segWidth) / sourceStepWidth
	scaledMin := maxInt(1, int(math.Round(float64(minFreq)*scale)))
	scaledMax := maxInt(scaledMin, int(math.Round(float64(maxFreq)*scale)))
	scaledAmp := minAmp * math.Max(0.2, float64(height)/7.0) * math.Max(0.2, scale)
	return (peakFreq >= scaledMin && peakFreq <= scaledMax) || peakAmp >= scaledAmp
}

func sampleGrayMeanStd(img image.Image, box maa.Rect) (float64, float64) {
	if img == nil || box[2] <= 0 || box[3] <= 0 {
		return 0, 0
	}
	bounds := img.Bounds()
	x1 := maxInt(bounds.Min.X, box[0])
	y1 := maxInt(bounds.Min.Y, box[1])
	x2 := minInt(bounds.Max.X, box[0]+box[2])
	y2 := minInt(bounds.Max.Y, box[1]+box[3])
	if x2 <= x1 || y2 <= y1 {
		return 0, 0
	}
	total := 0.0
	sum := 0.0
	sumSq := 0.0
	for y := y1; y < y2; y++ {
		for x := x1; x < x2; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8, g8, b8 := float64(r>>8), float64(g>>8), float64(b>>8)
			gray := 0.299*r8 + 0.587*g8 + 0.114*b8
			sum += gray
			sumSq += gray * gray
			total++
		}
	}
	if total == 0 {
		return 0, 0
	}
	mean := sum / total
	variance := sumSq/total - mean*mean
	if variance < 0 {
		variance = 0
	}
	return mean, math.Sqrt(variance)
}

func sampleStripeFillPct(img image.Image, box maa.Rect, rMin, rMax, gMin, gMax, bMin, bMax uint32) float64 {
	if img == nil || box[2] <= 0 || box[3] <= 0 {
		return 0
	}
	bounds := img.Bounds()
	x1 := maxInt(bounds.Min.X, box[0])
	y1 := maxInt(bounds.Min.Y, box[1])
	x2 := minInt(bounds.Max.X, box[0]+box[2])
	y2 := minInt(bounds.Max.Y, box[1]+box[3])
	if x2 <= x1 || y2 <= y1 {
		return 0
	}

	width := x2 - x1
	height := y2 - y1
	mask := make([][]bool, height)
	hasWhite := false
	for y := 0; y < height; y++ {
		mask[y] = make([]bool, width)
	}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			r, g, b, _ := img.At(x1+x, y1+y).RGBA()
			r8, g8, b8 := r>>8, g>>8, b>>8
			if r8 >= rMin && r8 <= rMax && g8 >= gMin && g8 <= gMax && b8 >= bMin && b8 <= bMax {
				mask[y][x] = true
				hasWhite = true
			}
		}
	}
	if !hasWhite {
		return 0
	}

	startX, endX := detectStripeRegion(mask)
	if startX == -2 {
		return -1
	}
	if startX == -1 {
		fallback := sampleColorPct(img, box, rMin, rMax, gMin, gMax, bMin, bMax) * 2
		if fallback > 1 {
			return 1
		}
		return fallback
	}
	return float64(endX-startX) / float64(width)
}

func detectStripeRegion(mask [][]bool) (int, int) {
	height := len(mask)
	if height == 0 {
		return -1, -1
	}
	width := len(mask[0])
	if width < 64 {
		return -1, -1
	}

	whiteCount := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if mask[y][x] {
				whiteCount++
			}
		}
	}
	if whiteCount == 0 {
		return 0, 0
	}

	cleaned := removeShortStripeRuns(mask, maxInt(1, int(float64(width)*0.009)))
	winW := maxInt(8, int(float64(width)*0.045))
	step := maxInt(1, int(float64(width)*0.012))

	var scores []float64
	var positions [][2]int
	for x := 0; x+winW <= width; x += step {
		windowWhite := 0
		total := height * winW
		for y := 0; y < height; y++ {
			for xx := x; xx < x+winW; xx++ {
				if cleaned[y][xx] {
					windowWhite++
				}
			}
		}
		whiteRatio := float64(windowWhite) / float64(total)
		score := 0.0
		if whiteRatio >= 0.05 && whiteRatio <= 0.75 {
			profile := make([]float64, winW)
			for xx := 0; xx < winW; xx++ {
				sum := 0.0
				for y := 0; y < height; y++ {
					if cleaned[y][x+xx] {
						sum++
					}
				}
				profile[xx] = sum
			}
			score = fftPeak(profile)
		}
		scores = append(scores, score)
		positions = append(positions, [2]int{x, x + winW})
	}

	if len(scores) == 0 {
		return -1, -1
	}
	maxScore := 0.0
	for _, score := range scores {
		if score > maxScore {
			maxScore = score
		}
	}
	if maxScore < 1e-3 {
		return 0, 0
	}
	fftThresh := math.Max(1.5, maxScore*0.45)

	failCount := 0
	endIdx := 0
	for i, score := range scores {
		if score >= fftThresh {
			endIdx = i
			failCount = 0
			continue
		}
		a, b := positions[i][0], positions[i][1]
		windowWhite := 0
		total := height * (b - a)
		for y := 0; y < height; y++ {
			for xx := a; xx < b; xx++ {
				if cleaned[y][xx] {
					windowWhite++
				}
			}
		}
		whiteRatio := float64(windowWhite) / float64(total)
		if whiteRatio < 0.375 {
			for _, later := range scores[i+1:] {
				if later > fftThresh {
					return -2, -2
				}
			}
			failCount++
			if failCount >= 1 {
				break
			}
		}
	}

	if endIdx == 0 {
		return 0, 0
	}
	if positions[endIdx][1]+step >= width {
		return 0, width
	}
	return 0, positions[endIdx][0]
}

func removeShortStripeRuns(mask [][]bool, threshold int) [][]bool {
	if threshold < 1 || len(mask) == 0 {
		return mask
	}
	height := len(mask)
	width := len(mask[0])
	result := make([][]bool, height)
	for y := 0; y < height; y++ {
		result[y] = make([]bool, width)
		copy(result[y], mask[y])
		x := 0
		for x < width {
			if !mask[y][x] {
				x++
				continue
			}
			start := x
			for x < width && mask[y][x] {
				x++
			}
			if x-start < threshold {
				for i := start; i < x; i++ {
					result[y][i] = false
				}
			}
		}
	}
	return result
}

func fftPeak(profile []float64) float64 {
	n := len(profile)
	if n < 2 {
		return 0
	}
	mean := 0.0
	for _, v := range profile {
		mean += v
	}
	mean /= float64(n)
	maxAmp := 0.0
	for k := 1; k < n/2; k++ {
		realPart := 0.0
		imagPart := 0.0
		for i, v := range profile {
			adj := v - mean
			angle := 2 * math.Pi * float64(k*i) / float64(n)
			realPart += adj * math.Cos(angle)
			imagPart -= adj * math.Sin(angle)
		}
		amp := math.Hypot(realPart, imagPart)
		if amp > maxAmp {
			maxAmp = amp
		}
	}
	return maxAmp
}

func fftPeakWithFrequency(profile []float64) (int, float64) {
	n := len(profile)
	if n < 2 {
		return 0, 0
	}
	mean := 0.0
	for _, v := range profile {
		mean += v
	}
	mean /= float64(n)
	maxAmp := 0.0
	maxFreq := 0
	for k := 1; k < n; k++ {
		realPart := 0.0
		imagPart := 0.0
		for i, v := range profile {
			adj := v - mean
			angle := 2 * math.Pi * float64(k*i) / float64(n)
			realPart += adj * math.Cos(angle)
			imagPart -= adj * math.Sin(angle)
		}
		amp := math.Hypot(realPart, imagPart)
		if amp > maxAmp {
			maxAmp = amp
			maxFreq = k
		}
	}
	return maxFreq, maxAmp
}

type combatOCRItem struct {
	text string
	box  maa.Rect
}

// refreshCooldowns ports ok-ww BaseCombatTask.refresh_cd(): OCR reads decimal
// cooldown labels in the lower-right skill strip, then maps them by x-position.
func (sa *ScreenAnalyzer) refreshCooldowns(ctx *maa.Context, img image.Image) {
	sa.ResonanceCD = false
	sa.EchoCD = false
	sa.LiberationCD = false
	detail, err := ctx.RunRecognition("Combat_CooldownOCR", img)
	if err != nil || detail == nil {
		return
	}
	for _, item := range combatOCRItems(detail) {
		text := normalizeCombatCDText(item.text)
		for _, token := range combatCDPattern.FindAllString(text, -1) {
			value, parseErr := strconv.ParseFloat(token, 64)
			if parseErr != nil || value <= 0.2 {
				continue
			}
			x := item.box[0]
			if x <= 0 {
				x = detail.Box[0]
			}
			switch {
			case x < 1101:
				sa.ResonanceCD = true
			case x > 1165:
				sa.LiberationCD = true
			default:
				sa.EchoCD = true
			}
		}
	}
}

func combatOCRItems(detail *maa.RecognitionDetail) []combatOCRItem {
	if detail == nil {
		return nil
	}
	if detail.Results == nil {
		return []combatOCRItem{{text: detail.DetailJson, box: detail.Box}}
	}
	results := detail.Results.Filtered
	if len(results) == 0 {
		results = detail.Results.All
	}
	items := make([]combatOCRItem, 0, len(results))
	for _, result := range results {
		ocr, ok := result.AsOCR()
		if !ok || ocr == nil {
			continue
		}
		items = append(items, combatOCRItem{text: ocr.Text, box: ocr.Box})
	}
	if len(items) == 0 {
		items = append(items, combatOCRItem{text: detail.DetailJson, box: detail.Box})
	}
	return items
}

func normalizeCombatCDText(text string) string {
	text = strings.TrimSpace(text)
	replacer := strings.NewReplacer(
		"：", ".",
		":", ".",
		"，", ".",
		",", ".",
		"·", ".",
		"。", ".",
		"O", "0",
		"o", "0",
		"I", "1",
		"l", "1",
		"|", "1",
		" ", "",
	)
	return replacer.Replace(text)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

package combat

import (
	"fmt"
	"image"
	"math"
	"time"

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
	CamellyaBudding  bool
	ChangliFortePct  float64
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

// Update runs all recognition nodes against one frame.
func (sa *ScreenAnalyzer) Update(ctx *maa.Context, img image.Image) bool {
	if img == nil {
		return false
	}

	// 1. Target lock (ok-ww: has_target, threshold 0.6)
	detail, err := ctx.RunRecognition("__Combat_Target", img, `{
		"__Combat_Target": {
			"recognition": "TemplateMatch",
			"template": "has_target.png",
			"threshold": 0.6
		}
	}`)
	sa.HasTarget = err == nil && detail != nil && detail.Hit

	// 2. Dodge prompt (ok-ww: dodge_prompt, threshold 0.6)
	detail, err = ctx.RunRecognition("__Combat_Dodge", img, `{
		"__Combat_Dodge": {
			"recognition": "TemplateMatch",
			"template": "dodge_prompt.png",
			"threshold": 0.6,
			"roi": [500, 300, 280, 420]
		}
	}`)
	sa.HasDodge = err == nil && detail != nil && detail.Hit

	// 3. Red HP bar via ColorMatch (ok-ww: enemy_health_color_red)
	detail, err = ctx.RunRecognition("__Combat_HP", img, `{
		"__Combat_HP": {
			"recognition": "ColorMatch",
			"lower": [55, 55, 174],
			"upper": [76, 85, 225],
			"min_width": 12,
			"min_height": 4
		}
	}`)
	sa.HasHPBar = err == nil && detail != nil && detail.Hit

	// 4. Boss HP bar
	detail, err = ctx.RunRecognition("__Combat_BossHP", img, `{
		"__Combat_BossHP": {
			"recognition": "ColorMatch",
			"lower": [4, 30, 245],
			"upper": [75, 185, 255],
			"roi": [360, 10, 560, 60]
		}
	}`)
	sa.HasBossHP = err == nil && detail != nil && detail.Hit

	// 5. F pickup prompt
	detail, err = ctx.RunRecognition("__Combat_Pick", img, `{
		"__Combat_Pick": {
			"recognition": "TemplateMatch",
			"template": "pick_up_f_hcenter_vcenter.png",
			"threshold": 0.65
		}
	}`)
	sa.PickupF = err == nil && detail != nil && detail.Hit

	// 6. Liberation availability hint.
	detail, err = ctx.RunRecognition("__Combat_LiberationReady", img, `{
		"__Combat_LiberationReady": {
			"recognition": "TemplateMatch",
			"template": "box_liberation.png",
			"threshold": 0.6,
			"roi": [1000, 500, 280, 220]
		}
	}`)
	sa.Liberation = err == nil && detail != nil && detail.Hit

	sa.RingElement = ringElementUnknown
	ringChecks := []struct {
		name     string
		template string
		element  int
	}{
		{"__Combat_RingConFullHavoc", "con_full_havoc.png", ringElementHavoc},
		{"__Combat_RingConFullSpectro", "con_full_spectro.png", ringElementSpectro},
		{"__Combat_RingConFullWind", "con_full_wind.png", ringElementWind},
		{"__Combat_RingConHavoc", "con_havoc.png", ringElementHavoc},
		{"__Combat_RingConSpectro", "con_spectro.png", ringElementSpectro},
		{"__Combat_RingConWind", "con_wind.png", ringElementWind},
		{"__Combat_RingLibReadyHavoc", "lib_ready_havoc.png", ringElementHavoc},
		{"__Combat_RingLibReadySpectro", "lib_ready_spectro.png", ringElementSpectro},
		{"__Combat_RingLibReadyWind", "lib_ready_wind.png", ringElementWind},
	}
	for _, rc := range ringChecks {
		detail, err = ctx.RunRecognition(rc.name, img, `{
			"`+rc.name+`": {
				"recognition": "TemplateMatch",
				"template": "`+rc.template+`",
				"threshold": 0.7
			}
		}`)
		if err == nil && detail != nil && detail.Hit {
			sa.RingElement = rc.element
			break
		}
	}

	// 6.5. Shared UI percentages/state used by many ok-ww character strategies.
	sa.ResonancePct = sampleNearWhitePct(img, maa.Rect{1062, 626, 45, 38})
	sa.EchoPct = sampleNearWhitePct(img, maa.Rect{1124, 626, 45, 39})
	sa.LiberationPct = sampleNearWhitePct(img, maa.Rect{1190, 626, 42, 38})
	sa.FortePct = sampleNearWhitePct(img, maa.Rect{750, 664, 20, 8})
	sa.Flying = sampleNearWhitePct(img, maa.Rect{1005, 629, 27, 27}) < 0.1
	sa.ZhezhiBluePct = sampleColorPct(img, maa.Rect{560, 666, 15, 7}, 160, 180, 240, 255, 245, 255)
	sa.PhoebeStarLight = sampleColorPct(img, maa.Rect{630, 670, 8, 7}, 240, 255, 240, 255, 180, 220)
	sa.PhoebeStarBlue = sampleColorPct(img, maa.Rect{630, 670, 8, 7}, 140, 200, 210, 255, 240, 255)
	sa.PhoebeRingBlue = sampleColorPct(img, maa.Rect{1055, 618, 54, 54}, 140, 200, 210, 255, 240, 255)
	sa.CamellyaRedPct = sampleColorPct(img, maa.Rect{1055, 618, 54, 54}, 180, 255, 20, 120, 20, 120)
	sa.ChangliFortePct = sampleStripeFillPct(img, maa.Rect{544, 668, 176, 4}, 240, 255, 85, 105, 95, 115)
	sa.CamellyaFortePct = sampleStripeFillPct(img, maa.Rect{543, 667, 182, 2}, 193, 255, 46, 93, 127, 163)
	sa.CamellyaBudPct = sampleStripeFillPct(img, maa.Rect{543, 667, 182, 2}, 220, 255, 161, 213, 168, 225)
	sa.ZaniFortePct = sampleColorPct(img, maa.Rect{543, 665, 185, 3}, 239, 255, 222, 255, 156, 196)
	sa.ZaniNightfallPct = sampleColorPct(img, maa.Rect{926, 616, 55, 56}, 210, 255, 215, 255, 180, 255)
	sa.ZaniBlazesPct = sampleColorPct(img, maa.Rect{543, 671, 183, 3}, 240, 255, 210, 255, 150, 220)
	sa.LinnaiColorPct = sampleNearWhitePct(img, maa.Rect{711, 650, 11, 22})

	detail, err = ctx.RunRecognition("__Combat_ZaniNotLiber", img, `{
		"__Combat_ZaniNotLiber": {
			"recognition": "TemplateMatch",
			"template": "box_target_enemy_inner.png",
			"threshold": 0.75,
			"roi": [954, 637, 49, 48]
		}
	}`)
	sa.ZaniNotLiberBox = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_ZaniLiber", img, `{
		"__Combat_ZaniLiber": {
			"recognition": "TemplateMatch",
			"template": "box_target_enemy_inner.png",
			"threshold": 0.75,
			"roi": [889, 636, 52, 49]
		}
	}`)
	sa.ZaniLiberBox = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_MouseForte", img, `{
		"__Combat_MouseForte": {
			"recognition": "TemplateMatch",
			"template": "mouse_forte.png",
			"threshold": 0.6,
			"roi": [360, 300, 80, 80]
		}
	}`)
	sa.MouseForteFull = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_EForte", img, `{
		"__Combat_EForte": {
			"recognition": "TemplateMatch",
			"template": "e_forte.png",
			"threshold": 0.6,
			"roi": [730, 650, 60, 40]
		}
	}`)
	sa.EForteFull = err == nil && detail != nil && detail.Hit
	sa.ForteFull = sa.FortePct > 0.08 || sa.EForteFull

	detail, err = ctx.RunRecognition("__Combat_LongAction1", img, `{
		"__Combat_LongAction1": {
			"recognition": "TemplateMatch",
			"template": "box_target_enemy_long.png",
			"threshold": 0.6,
			"roi": [860, 615, 80, 50]
		}
	}`)
	sa.HasLongAction = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_LongAction2", img, `{
		"__Combat_LongAction2": {
			"recognition": "TemplateMatch",
			"template": "target_box_long2.png",
			"threshold": 0.6,
			"roi": [820, 615, 70, 50]
		}
	}`)
	sa.HasLongAction2 = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_HiyukiLibForte", img, `{
		"__Combat_HiyukiLibForte": {
			"recognition": "TemplateMatch",
			"template": "hiyuki_lib_forte.png",
			"threshold": 0.7
		}
	}`)
	sa.HiyukiLibForte = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_HiyukiLeft", img, `{
		"__Combat_HiyukiLeft": {
			"recognition": "TemplateMatch",
			"template": "hiyuki_left.png",
			"threshold": 0.5
		}
	}`)
	sa.HiyukiLeft = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_HiyukiRight", img, `{
		"__Combat_HiyukiRight": {
			"recognition": "TemplateMatch",
			"template": "hiyuki_right.png",
			"threshold": 0.5
		}
	}`)
	sa.HiyukiRight = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_AemeathEnhanceE", img, `{
		"__Combat_AemeathEnhanceE": {
			"recognition": "TemplateMatch",
			"template": "aemeath_e1.png",
			"threshold": 0.7
		}
	}`)
	enhance1 := err == nil && detail != nil && detail.Hit
	detail, err = ctx.RunRecognition("__Combat_AemeathEnhanceE2", img, `{
		"__Combat_AemeathEnhanceE2": {
			"recognition": "TemplateMatch",
			"template": "aemeath_e2.png",
			"threshold": 0.7
		}
	}`)
	enhance2 := err == nil && detail != nil && detail.Hit
	sa.AemeathEnhanceE = enhance1 || enhance2

	detail, err = ctx.RunRecognition("__Combat_AemeathLib2", img, `{
		"__Combat_AemeathLib2": {
			"recognition": "TemplateMatch",
			"template": "aemeath_lib2.png",
			"threshold": 0.7
		}
	}`)
	sa.AemeathLib2 = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_LupaWolfReady", img, `{
		"__Combat_LupaWolfReady": {
			"recognition": "TemplateMatch",
			"template": "lupa_wolf_icon2.png",
			"threshold": 0.85
		}
	}`)
	sa.LupaWolfReady = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_CartethyiaSword1", img, `{
		"__Combat_CartethyiaSword1": {
			"recognition": "TemplateMatch",
			"template": "forte_cartethyia_sword1.png",
			"threshold": 0.9
		}
	}`)
	sa.CartethyiaSword1 = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_CartethyiaSword2", img, `{
		"__Combat_CartethyiaSword2": {
			"recognition": "TemplateMatch",
			"template": "forte_cartethyia_sword2.png",
			"threshold": 0.9
		}
	}`)
	sa.CartethyiaSword2 = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_CartethyiaSword3", img, `{
		"__Combat_CartethyiaSword3": {
			"recognition": "TemplateMatch",
			"template": "forte_cartethyia_sword3.png",
			"threshold": 0.9
		}
	}`)
	sa.CartethyiaSword3 = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_CartethyiaBigLib", img, `{
		"__Combat_CartethyiaBigLib": {
			"recognition": "TemplateMatch",
			"template": "lib_cartethyia_big.png",
			"threshold": 0.6
		}
	}`)
	sa.CartethyiaBigLib = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_CartethyiaSmall", img, `{
		"__Combat_CartethyiaSmall": {
			"recognition": "TemplateMatch",
			"template": "forte_cartethyia_sword3.png",
			"threshold": 0.5
		}
	}`)
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

	detail, err = ctx.RunRecognition("__Combat_LuhesiKickReady", img, `{
		"__Combat_LuhesiKickReady": {
			"recognition": "TemplateMatch",
			"template": "luhesi_kick.png",
			"threshold": 0.7
		}
	}`)
	sa.LuhesiKickReady = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_LuhesiLibReady", img, `{
		"__Combat_LuhesiLibReady": {
			"recognition": "TemplateMatch",
			"template": "box_luhesi_lib.png",
			"threshold": 0.7
		}
	}`)
	sa.LuhesiLibReady = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_IunoHeavyReady", img, `{
		"__Combat_IunoHeavyReady": {
			"recognition": "TemplateMatch",
			"template": "iuno_heavy.png",
			"threshold": 0.6
		}
	}`)
	sa.IunoHeavyReady = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_IunoJumpReady", img, `{
		"__Combat_IunoJumpReady": {
			"recognition": "TemplateMatch",
			"template": "iuno_jump.png",
			"threshold": 0.6
		}
	}`)
	sa.IunoJumpReady = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_AugustaLibReady", img, `{
		"__Combat_AugustaLibReady": {
			"recognition": "TemplateMatch",
			"template": "Augusta_lib1.png",
			"threshold": 0.5
		}
	}`)
	sa.AugustaLibReady = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_AugustaMajesty", img, `{
		"__Combat_AugustaMajesty": {
			"recognition": "TemplateMatch",
			"template": "Augusta_lib2.png",
			"threshold": 0.5
		}
	}`)
	sa.AugustaMajesty = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_AugustaProwess", img, `{
		"__Combat_AugustaProwess": {
			"recognition": "TemplateMatch",
			"template": "target_enemy_long_inner.png",
			"threshold": 0.8
		}
	}`)
	sa.AugustaProwess = err == nil && detail != nil && detail.Hit

	detail, err = ctx.RunRecognition("__Combat_CamellyaBudding", img, `{
		"__Combat_CamellyaBudding": {
			"recognition": "TemplateMatch",
			"template": "camellya_budding.png",
			"threshold": 0.7
		}
	}`)
	sa.CamellyaBudding = err == nil && detail != nil && detail.Hit

	// 7. Character portraits
	sa.ConcertoPct = 0
	existCount := 0
	currentIdx := -1
	for i := range 3 {
		tpl := []string{"char_1_text.png", "char_2_text.png", "char_3_text.png"}[i]
		detail, err = ctx.RunRecognition("__Combat_Char"+string(rune('1'+i)), img, `{
			"__Combat_Char`+string(rune('1'+i))+`": {
				"recognition": "TemplateMatch",
				"template": "`+tpl+`",
				"threshold": 0.7
			}
		}`)
		textHit := err == nil && detail != nil && detail.Hit
		if textHit {
			existCount++
		} else if currentIdx < 0 {
			currentIdx = i
		}

		conTpl := []string{"con_mark_char_1.png", "con_mark_char_2.png", "con_mark_char_3.png"}[i]
		detail, err = ctx.RunRecognition("__Combat_Con"+string(rune('1'+i)), img, `{
			"__Combat_Con`+string(rune('1'+i))+`": {
				"recognition": "TemplateMatch",
				"template": "`+conTpl+`",
				"threshold": 0.6
			}
		}`)
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
		Float64("zhezhi_blue", sa.ZhezhiBluePct).
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
	bestScore := 0.0
	bestMeta := charMeta{}
	for _, meta := range charTemplates {
		nodeName := fmt.Sprintf("__Combat_CharName_%d_%s", index+1, meta.Name)
		detail, err := ctx.RunRecognition(
			nodeName,
			img,
			fmt.Sprintf(`{
				%q: {
					"recognition": "TemplateMatch",
					"template": %q,
					"threshold": 0.55,
					"roi": [%d, %d, %d, %d]
				}
			}`, nodeName, meta.Template, box[0], box[1], box[2], box[3]),
		)
		if err != nil || detail == nil || !detail.Hit {
			continue
		}
		score := 0.6
		if detail.Results != nil && detail.Results.Best != nil {
			if tm, ok := detail.Results.Best.AsTemplateMatch(); ok {
				score = tm.Score
			}
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
	template := []string{"box_char_1.png", "box_char_2.png", "box_char_3.png"}[index]
	nodeName := fmt.Sprintf("__Combat_BoxChar_%d", index+1)
	detail, err := ctx.RunRecognition(
		nodeName,
		img,
		fmt.Sprintf(`{
			%q: {
				"recognition": "TemplateMatch",
				"template": %q,
				"threshold": 0.5
			}
		}`, nodeName, template),
	)
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

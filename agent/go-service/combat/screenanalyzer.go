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
	HasTarget   bool
	HasDodge    bool
	HasHPBar    bool
	HasBossHP   bool
	PickupF     bool
	Liberation  bool
	CharAlive   [3]bool
	CharSlots   [3]charSlot
	CurrentIdx  int
	TeamSize    int
	ConcertoPct float64
	RingElement int // -1 = unknown

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
		Int("current", sa.CurrentIdx+1).Int("team_size", sa.TeamSize).
		Float64("concerto", sa.ConcertoPct).Msg("frame")

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

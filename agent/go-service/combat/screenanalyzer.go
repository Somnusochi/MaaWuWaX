package combat

import (
	"image"

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
	CharAlive   [3]bool
	ConcertoPct float64
	RingElement int // -1 = unknown
}

var screenAnalyzer = &ScreenAnalyzer{RingElement: -1}

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

	// 6. Character portraits
	for i := range 3 {
		tpl := []string{"char_1_text.png", "char_2_text.png", "char_3_text.png"}[i]
		detail, err = ctx.RunRecognition("__Combat_Char"+string(rune('1'+i)), img, `{
			"__Combat_Char`+string(rune('1'+i))+`": {
				"recognition": "TemplateMatch",
				"template": "`+tpl+`",
				"threshold": 0.7
			}
		}`)
		sa.CharAlive[i] = err == nil && detail != nil && detail.Hit
	}

	log.Debug().Str("component", "ScreenAnalyzer").
		Bool("target", sa.HasTarget).Bool("hp", sa.HasHPBar).
		Bool("boss", sa.HasBossHP).Bool("dodge", sa.HasDodge).Msg("frame")

	return sa.InCombat()
}

// InCombat — ok-ww: has_target() OR check_health_bar()
func (sa *ScreenAnalyzer) InCombat() bool {
	return sa.HasTarget || sa.HasHPBar || sa.HasBossHP
}

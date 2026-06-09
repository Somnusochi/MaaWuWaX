// Package rogue implements half-auto rogue Custom Actions for Wuthering Waves.
package rogue

import (
	"strings"
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// RogueMainAction — orchestrates the rogue loop: fight → explore → buff select.
// ---------------------------------------------------------------------------

type RogueMainAction struct{}

var _ maa.CustomActionRunner = &RogueMainAction{}

func (a *RogueMainAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	log.Info().Str("component", "RogueMain").Msg("rogue loop started")

	for {
		if ctx.GetTasker().Stopping() {
			return true
		}

		// Check for challenge end.
		detail, err := ctx.RunRecognition(
			"__Rogue_ChallengeEnd",
			nil,
			`{
				"__Rogue_ChallengeEnd": {
					"recognition": "Or",
					"any_of": [
						{"recognition": "OCR", "expected": "挑战结束"},
						{"recognition": "OCR", "expected": "Challenge End"}
					]
				}
			}`,
		)
		if err == nil && detail != nil && detail.Hit {
			log.Info().Str("component", "RogueMain").Msg("challenge ended")
			return true
		}

		// Check for in-realm state — not in team means we're in UI.
		inTeamDetail, err := ctx.RunRecognition(
			"__Rogue_InTeam",
			nil,
			`{
				"__Rogue_InTeam": {
					"recognition": "TemplateMatch",
					"template": "minimap.png",
					"threshold": 0.7,
					"roi": [1050, 20, 200, 160]
				}
			}`,
		)
		if err != nil || inTeamDetail == nil || !inTeamDetail.Hit {
			// Not in team — handle UI states.
			a.handleRogueUI(ctx)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// In team — try combat.
		combatDetail, err := ctx.RunRecognition(
			"__Rogue_HasTarget",
			nil,
			`{
				"__Rogue_HasTarget": {
					"recognition": "TemplateMatch",
					"template": "has_target.png",
					"threshold": 0.6
				}
			}`,
		)
		if err == nil && combatDetail != nil && combatDetail.Hit {
			log.Debug().Str("component", "RogueMain").Msg("engaging combat")
			ctx.RunAction("Rogue_Fight", maa.Rect{0, 0, 1, 1}, "", nil)
			continue
		}

		// No target — press F or walk forward.
		fDetail, err := ctx.RunRecognition(
			"__Rogue_PressF",
			nil,
			`{
				"__Rogue_PressF": {
					"recognition": "TemplateMatch",
					"template": "pick_up_f.png",
					"threshold": 0.6
				}
			}`,
		)
		if err == nil && fDetail != nil && fDetail.Hit {
			ctrl := ctx.GetTasker().GetController()
			ctrl.PostClickKey(3).Wait()
			time.Sleep(1000 * time.Millisecond)
			continue
		}

		// Walk forward briefly.
		ctrl := ctx.GetTasker().GetController()
		wCode := int32(13) // W key
		ctrl.PostKeyDown(wCode).Wait()
		time.Sleep(800 * time.Millisecond)
		ctrl.PostKeyUp(wCode).Wait()
		time.Sleep(200 * time.Millisecond)
	}
}

func (a *RogueMainAction) handleRogueUI(ctx *maa.Context) {
	// Trade UI — skip.
	tradeDetail, _ := ctx.RunRecognition(
		"__Rogue_Trade",
		nil,
		`{
			"__Rogue_Trade": {
				"recognition": "OCR",
				"expected": "交易",
				"roi": [10, 20, 180, 80]
			}
		}`,
	)
	if tradeDetail != nil && tradeDetail.Hit {
		ctrl := ctx.GetTasker().GetController()
		ctrl.PostClickKey(53).Wait() // ESC
		time.Sleep(2000 * time.Millisecond)
		return
	}

	// Buff select.
	buffDetail, _ := ctx.RunRecognition(
		"__Rogue_BuffSelect",
		nil,
		`{
			"__Rogue_BuffSelect": {
				"recognition": "OCR",
				"expected": "隐喻获得"
			}
		}`,
	)
	if buffDetail != nil && buffDetail.Hit {
		ctx.RunAction("Rogue_BuffSelect", maa.Rect{0, 0, 1, 1}, "", nil)
		return
	}

	// Gain echo — dismiss.
	gainDetail, _ := ctx.RunRecognition(
		"__Rogue_GainEcho",
		nil,
		`{
			"__Rogue_GainEcho": {
				"recognition": "OCR",
				"expected": "获得",
				"roi": [550, 130, 180, 80]
			}
		}`,
	)
	if gainDetail != nil && gainDetail.Hit {
		ctrl := ctx.GetTasker().GetController()
		ctrl.PostClick(640, 580).Wait()
		time.Sleep(2000 * time.Millisecond)
		return
	}

	// Continue explore.
	contDetail, _ := ctx.RunRecognition(
		"__Rogue_Continue",
		nil,
		`{
			"__Rogue_Continue": {
				"recognition": "OCR",
				"expected": "退出确认"
			}
		}`,
	)
	if contDetail != nil && contDetail.Hit {
		ctrl := ctx.GetTasker().GetController()
		ctrl.PostClick(860, 440).Wait()
		time.Sleep(2000 * time.Millisecond)
		return
	}
}

// ---------------------------------------------------------------------------
// RogueBuffSelectAction — OCRs buff names and selects based on whitelist/blacklist.
// ---------------------------------------------------------------------------

type RogueBuffSelectAction struct{}

var _ maa.CustomActionRunner = &RogueBuffSelectAction{}

type rogueBuffParam struct {
	Blacklist []string `json:"blacklist"`
	Whitelist []string `json:"whitelist"`
}

func defaultRogueBuffParam() rogueBuffParam {
	return rogueBuffParam{
		Blacklist: []string{"雷暴", "旋风", "矛盾晶体"},
		Whitelist: []string{"心流", "悲鸣纪", "余音贝", "齿轮之心", "全知之眼", "指南针", "医疗箱"},
	}
}

func (a *RogueBuffSelectAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := defaultRogueBuffParam()
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "RogueBuffSelect").Msg("failed to parse param")
		}
	}

	ctrl := ctx.GetTasker().GetController()

	// OCR the buff area (3 buff choices in a row).
	detail, err := ctx.RunRecognition(
		"__RogueBuff_OCR",
		nil,
		`{
			"__RogueBuff_OCR": {
				"recognition": "OCR",
				"roi": [240, 395, 800, 60]
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		// Fallback: click middle buff.
		log.Warn().Str("component", "RogueBuffSelect").Msg("OCR failed, clicking middle")
		ctrl.PostClick(640, 430).Wait()
		time.Sleep(1000 * time.Millisecond)
		return true
	}

	text := detail.DetailJson
	// Try whitelist first.
	for _, w := range param.Whitelist {
		if strings.Contains(text, w) {
			log.Info().Str("component", "RogueBuffSelect").Str("buff", w).Msg("selected whitelist buff")
			ctrl.PostClick(640, 430).Wait()
			time.Sleep(1000 * time.Millisecond)
			return true
		}
	}

	// No whitelist hit, click middle (avoid blacklist).
	for _, b := range param.Blacklist {
		if strings.Contains(text, b) {
			log.Debug().Str("component", "RogueBuffSelect").Str("blacklist", b).Msg("avoiding blacklisted buff")
		}
	}

	ctrl.PostClick(640, 430).Wait()
	time.Sleep(1000 * time.Millisecond)
	return true
}

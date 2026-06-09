// Package daily implements daily-routine Custom Actions for Wuthering Waves.
package daily

import (
	"fmt"
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// SpendStaminaAction — delegates stamina spending to the appropriate sub-flow.
// Param: {"stamina_type": "tacet"|"forgery"|"simulation"}
// ---------------------------------------------------------------------------

type staminaParam struct {
	StaminaType string `json:"stamina_type"`
}

type SpendStaminaAction struct{}

var _ maa.CustomActionRunner = &SpendStaminaAction{}

func (a *SpendStaminaAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	var params staminaParam
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &params); err != nil {
			log.Warn().Err(err).Str("component", "SpendStamina").Msg("failed to parse param")
		}
	}

	log.Info().
		Str("component", "SpendStamina").
		Str("type", params.StaminaType).
		Msg("spending stamina")

	// Navigate to the stamina spending flow via Pipeline.
	switch params.StaminaType {
	case "tacet":
		ctx.RunAction("Daily_NavTacet", maa.Rect{0, 0, 1, 1}, "", nil)
	case "forgery":
		ctx.RunAction("Daily_NavForgery", maa.Rect{0, 0, 1, 1}, "", nil)
	case "simulation":
		ctx.RunAction("Daily_NavSimulation", maa.Rect{0, 0, 1, 1}, "", nil)
	default:
		log.Info().Str("component", "SpendStamina").Msg("no stamina type specified, skipping")
		return true
	}

	return true
}

// ---------------------------------------------------------------------------
// ClaimMailAction — navigates to mail page and claims all.
// ---------------------------------------------------------------------------

type ClaimMailAction struct{}

var _ maa.CustomActionRunner = &ClaimMailAction{}

func (a *ClaimMailAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	log.Info().Str("component", "ClaimMail").Msg("claiming mail")
	ctrl := ctx.GetTasker().GetController()

	// Open mail page: navigate via menu or shortcut.
	// In WuWa, mail is accessed from the terminal menu.
	// Press ESC to open menu, then navigate to mail tab.
	escCode := keycode.MustCode("ESC")
	ctrl.PostClickKey(escCode).Wait()
	time.Sleep(1000 * time.Millisecond)

	// Look for mail icon and click it.
	detail, err := ctx.RunRecognition(
		"__ClaimMail_FindIcon",
		nil, // Will use latest screenshot.
		`{
			"__ClaimMail_FindIcon": {
				"recognition": "OCR",
				"expected": "邮件",
				"roi": [0, 0, 400, 720]
			}
		}`,
	)
	if err == nil && detail != nil && detail.Hit {
		box := detail.Box
		cx := int32(box[0] + box[2]/2)
		cy := int32(box[1] + box[3]/2)
		ctrl.PostClick(cx, cy).Wait()
		time.Sleep(1000 * time.Millisecond)

		// Click "Claim All" button.
		claimDetail, err := ctx.RunRecognition(
			"__ClaimMail_ClaimAll",
			nil,
			`{
				"__ClaimMail_ClaimAll": {
					"recognition": "TemplateMatch",
					"template": "btn_claim_all.png",
					"threshold": 0.7
				}
			}`,
		)
		if err == nil && claimDetail != nil && claimDetail.Hit {
			ctrl.PostClick(
				int32(claimDetail.Box[0]+claimDetail.Box[2]/2),
				int32(claimDetail.Box[1]+claimDetail.Box[3]/2),
			).Wait()
			time.Sleep(1500 * time.Millisecond)
			log.Info().Str("component", "ClaimMail").Msg("mail claimed")
		}
	}

	// Close menu.
	ctrl.PostClickKey(escCode).Wait()
	time.Sleep(500 * time.Millisecond)
	return true
}

// ---------------------------------------------------------------------------
// ClaimBattlePassAction — opens battle pass and claims rewards.
// ---------------------------------------------------------------------------

type ClaimBattlePassAction struct{}

var _ maa.CustomActionRunner = &ClaimBattlePassAction{}

func (a *ClaimBattlePassAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	log.Info().Str("component", "ClaimBattlePass").Msg("claiming battle pass")
	ctrl := ctx.GetTasker().GetController()

	// Open battle pass via shortcut or navigation.
	// WuWa battle pass is typically accessed from the terminal menu.
	escCode := keycode.MustCode("ESC")
	ctrl.PostClickKey(escCode).Wait()
	time.Sleep(1000 * time.Millisecond)

	// Look for battle pass tab via OCR.
	bpDetail, err := ctx.RunRecognition(
		"__ClaimBP_FindTab",
		nil,
		`{
			"__ClaimBP_FindTab": {
				"recognition": "OCR",
				"expected": "先约电台",
				"roi": [0, 0, 400, 720]
			}
		}`,
	)
	if err == nil && bpDetail != nil && bpDetail.Hit {
		box := bpDetail.Box
		ctrl.PostClick(
			int32(box[0]+box[2]/2),
			int32(box[1]+box[3]/2),
		).Wait()
		time.Sleep(1000 * time.Millisecond)

		// Try to find and click claim buttons.
		for attempt := 0; attempt < 3; attempt++ {
			claimDetail, err := ctx.RunRecognition(
				fmt.Sprintf("__ClaimBP_Claim_%d", attempt),
				nil,
				`{
					"__ClaimBP_Claim": {
						"recognition": "OCR",
						"expected": "领取"
					}
				}`,
			)
			if err != nil || claimDetail == nil || !claimDetail.Hit {
				break
			}
			box := claimDetail.Box
			ctrl.PostClick(
				int32(box[0]+box[2]/2),
				int32(box[1]+box[3]/2),
			).Wait()
			time.Sleep(800 * time.Millisecond)
		}

		log.Info().Str("component", "ClaimBattlePass").Msg("battle pass claimed")
	}

	// Close menu.
	ctrl.PostClickKey(escCode).Wait()
	time.Sleep(500 * time.Millisecond)
	return true
}

// ---------------------------------------------------------------------------
// DailyProgressReader — OCR recognition for daily progress "/180".
// ---------------------------------------------------------------------------

type DailyProgressReader struct{}

var _ maa.CustomRecognitionRunner = &DailyProgressReader{}

func (r *DailyProgressReader) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	detail, err := ctx.RunRecognition(
		"__DailyProgress_OCR",
		arg.Img,
		`{
			"__DailyProgress_OCR": {
				"recognition": "OCR",
				"roi": [150, 100, 500, 100],
				"expected": "/180"
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		log.Debug().Str("component", "DailyProgress").Msg("progress not found")
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    detail.Box,
		Detail: detail.DetailJson,
	}, true
}

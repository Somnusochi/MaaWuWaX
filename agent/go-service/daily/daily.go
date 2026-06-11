// Package daily implements daily-routine Custom Actions for Wuthering Waves.
package daily

import (
	"fmt"
	"image"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

var dailyProgressRe = regexp.MustCompile(`(\d{1,3})\s*/\s*180`)
var dailyPointsRe = regexp.MustCompile(`\d+`)
var lastDailyProgress = -1
var lastDailyRewardReady = false

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
		Int("daily_progress", lastDailyProgress).
		Bool("daily_reward_ready", lastDailyRewardReady).
		Msg("spending stamina")

	if lastDailyProgress >= 180 || lastDailyRewardReady {
		log.Info().Str("component", "SpendStamina").Msg("daily reward already ready or progress full, skipping stamina")
		return true
	}

	// Run the selected stamina sub-pipeline synchronously via RunTask.
	// This blocks until the sub-pipeline completes, then control returns
	// to the Daily pipeline for claiming rewards.
	switch params.StaminaType {
	case "tacet":
		ctx.RunTask("Tacet_Main")
	case "forgery":
		ctx.RunTask("Forgery_Main")
	case "simulation":
		ctx.RunTask("Simulation_Main")
	default:
		log.Info().Str("component", "SpendStamina").Msg("no stamina type specified, skipping")
		return true
	}

	return true
}

// DailyNeedsStaminaRecognition gates stamina routing after daily progress has
// been read from the activity book.
type DailyNeedsStaminaRecognition struct{}

var _ maa.CustomRecognitionRunner = &DailyNeedsStaminaRecognition{}

func (r *DailyNeedsStaminaRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	if lastDailyProgress >= 180 || lastDailyRewardReady {
		log.Info().
			Str("component", "DailyNeedsStamina").
			Int("daily_progress", lastDailyProgress).
			Bool("daily_reward_ready", lastDailyRewardReady).
			Msg("skip stamina routing")
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: fmt.Sprintf(`{"daily_progress":%d,"reward_ready":%t,"needs_stamina":true}`, lastDailyProgress, lastDailyRewardReady),
	}, true
}

type nightmareModeParam struct {
	NightmareMode string `json:"nightmare_mode"`
	StaminaType   string `json:"stamina_type"`
}

// DailyNeedsNightmareRecognition gates optional NightmareNest routing.
type DailyNeedsNightmareRecognition struct{}

var _ maa.CustomRecognitionRunner = &DailyNeedsNightmareRecognition{}

func (r *DailyNeedsNightmareRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := nightmareModeParam{NightmareMode: "none"}
	if arg != nil && arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "DailyNeedsNightmare").Msg("failed to parse param")
		}
	}
	if param.NightmareMode == "" || param.NightmareMode == "none" {
		return nil, false
	}
	if param.NightmareMode == "capture" && param.StaminaType == "tacet" {
		log.Info().
			Str("component", "DailyNeedsNightmare").
			Str("mode", param.NightmareMode).
			Str("stamina_type", param.StaminaType).
			Msg("skip capture-mode nightmare because tacet route already covers daily echo")
		return nil, false
	}
	if lastDailyProgress >= 180 || lastDailyRewardReady {
		log.Info().
			Str("component", "DailyNeedsNightmare").
			Str("mode", param.NightmareMode).
			Str("stamina_type", param.StaminaType).
			Int("daily_progress", lastDailyProgress).
			Bool("daily_reward_ready", lastDailyRewardReady).
			Msg("skip nightmare routing")
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: fmt.Sprintf(`{"nightmare_mode":%q,"stamina_type":%q,"needs_nightmare":true}`, param.NightmareMode, param.StaminaType),
	}, true
}

type runNightmareNestParam struct {
	NightmareMode string `json:"nightmare_mode"`
}

// RunNightmareNestAction runs NightmareNest with lightweight pipeline overrides.
type RunNightmareNestAction struct{}

var _ maa.CustomActionRunner = &RunNightmareNestAction{}

func (a *RunNightmareNestAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := runNightmareNestParam{NightmareMode: "capture"}
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "RunNightmareNest").Msg("failed to parse param")
		}
	}

	patch := map[string]any{}
	switch param.NightmareMode {
	case "nightmare":
		patch["NightmareNest_ScrollMengyan"] = map[string]any{
			"on_error": []string{"NightmareNest_Done"},
		}
	case "canxiang":
		patch["NightmareNest_BookOpen"] = map[string]any{
			"next": []string{"NightmareNest_SelectCanxiang"},
		}
	case "capture":
		patch["NightmareNest_EchoNotification"] = map[string]any{
			"next": []string{"NightmareNest_Done"},
		}
		patch["NightmareNest_CollectEcho"] = map[string]any{
			"next": []string{"NightmareNest_Done"},
		}
	case "all":
		// default NightmareNest behavior already means both pages.
	}
	if len(patch) > 0 {
		if err := ctx.OverridePipeline(patch); err != nil {
			log.Warn().Err(err).Str("component", "RunNightmareNest").Msg("failed to override nightmare pipeline")
			return false
		}
	}

	detail, err := ctx.RunTask("NightmareNest_Main")
	if err != nil || detail == nil || !detail.Status.Success() {
		log.Warn().Err(err).Str("component", "RunNightmareNest").Str("mode", param.NightmareMode).Msg("NightmareNest run failed")
		return false
	}

	log.Info().Str("component", "RunNightmareNest").Str("mode", param.NightmareMode).Msg("NightmareNest run completed")
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
					"template": "claim_btn.png",
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
// ClaimDailyRewardsAction — claims daily activity rewards with template and
// coordinate fallbacks from the daily book page.
// ---------------------------------------------------------------------------

type ClaimDailyRewardsAction struct{}

var _ maa.CustomActionRunner = &ClaimDailyRewardsAction{}

func (a *ClaimDailyRewardsAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	log.Info().Str("component", "ClaimDailyRewards").Msg("claiming daily rewards")
	ctrl := ctx.GetTasker().GetController()

	if a.clickTemplate(ctx, "claim_btn.png", 0.7) {
		time.Sleep(1500 * time.Millisecond)
	}

	// OK-WW fallback: open the daily claim area, then click the 100-point box.
	ctrl.PostClick(1128, 171).Wait()
	time.Sleep(1200 * time.Millisecond)
	ctrl.PostClick(1190, 634).Wait()
	time.Sleep(1000 * time.Millisecond)

	for i := 0; i < 3; i++ {
		if !a.clickTemplate(ctx, "claim_btn.png", 0.7) {
			break
		}
		time.Sleep(800 * time.Millisecond)
	}

	ctrl.PostClickKey(keycode.MustCode("ESC")).Wait()
	time.Sleep(800 * time.Millisecond)
	return true
}

func (a *ClaimDailyRewardsAction) clickTemplate(ctx *maa.Context, template string, threshold float64) bool {
	detail, err := ctx.RunRecognition(
		"__ClaimDailyRewards_Button",
		nil,
		fmt.Sprintf(`{
			"__ClaimDailyRewards_Button": {
				"recognition": "TemplateMatch",
				"template": %q,
				"threshold": %.2f
			}
		}`, template, threshold),
	)
	if err != nil || detail == nil || !detail.Hit {
		return false
	}

	ctx.GetTasker().GetController().PostClick(
		int32(detail.Box[0]+detail.Box[2]/2),
		int32(detail.Box[1]+detail.Box[3]/2),
	).Wait()
	return true
}

// ---------------------------------------------------------------------------
// DailyProgressReader — OCR recognition for daily progress "/180".
// ---------------------------------------------------------------------------

type DailyProgressReader struct{}

var _ maa.CustomRecognitionRunner = &DailyProgressReader{}

func (r *DailyProgressReader) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	for attempt := 0; attempt < 2; attempt++ {
		name := fmt.Sprintf("__DailyProgress_OCR_%d", attempt)
		detail, err := ctx.RunRecognition(
			name,
			arg.Img,
			fmt.Sprintf(`{
				%q: {
					"recognition": "OCR",
					"roi": [150, 100, 500, 440],
					"expected": "/180"
				}
			}`, name),
		)
		if err == nil && detail != nil && detail.Hit {
			lastDailyProgress = parseDailyProgress(detail.DetailJson)
			lastDailyRewardReady = readDailyRewardReady(ctx, arg.Img)
			log.Info().
				Str("component", "DailyProgress").
				Int("progress", lastDailyProgress).
				Bool("reward_ready", lastDailyRewardReady).
				Msg("daily progress read")
			return &maa.CustomRecognitionResult{
				Box:    detail.Box,
				Detail: fmt.Sprintf(`{"progress":%d,"reward_ready":%t}`, lastDailyProgress, lastDailyRewardReady),
			}, true
		}

		ctx.GetTasker().GetController().PostClick(1247, 432).Wait()
		time.Sleep(1000 * time.Millisecond)
	}

	log.Debug().Str("component", "DailyProgress").Msg("progress not found")
	return nil, false
}

func parseDailyProgress(text string) int {
	text = strings.Trim(text, `"`)
	match := dailyProgressRe.FindStringSubmatch(text)
	if len(match) < 2 {
		return -1
	}
	progress, err := strconv.Atoi(match[1])
	if err != nil {
		return -1
	}
	return progress
}

func readDailyRewardReady(ctx *maa.Context, img image.Image) bool {
	detail, err := ctx.RunRecognition(
		"__DailyRewardPoints_OCR",
		img,
		`{
			"__DailyRewardPoints_OCR": {
				"recognition": "OCR",
				"roi": [243, 576, 141, 94]
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		return false
	}
	match := dailyPointsRe.FindString(detail.DetailJson)
	if match == "" {
		return false
	}
	points, err := strconv.Atoi(match)
	if err != nil {
		return false
	}
	return points >= 100
}

// ---------------------------------------------------------------------------
// Multi-account state (package-level, shared across action instances).
// ---------------------------------------------------------------------------

type multiAccountParam struct {
	MaxAccounts int `json:"max_accounts"`
}

var (
	multiAccountDone   int
	multiAccountFailed int
)

// ---------------------------------------------------------------------------
// MultiAccountSwitchAction — switches to the next account in a multi-account
// run. Uses package-level counters to track progress across invocations.
// ---------------------------------------------------------------------------

type MultiAccountSwitchAction struct{}

var _ maa.CustomActionRunner = &MultiAccountSwitchAction{}

func (a *MultiAccountSwitchAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := multiAccountParam{MaxAccounts: 2}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "MultiAccountSwitch").Msg("failed to parse param")
		}
	}

	multiAccountDone++
	log.Info().
		Str("component", "MultiAccountSwitch").
		Int("done", multiAccountDone).
		Int("failed", multiAccountFailed).
		Int("max", param.MaxAccounts).
		Msg("account finished")

	if multiAccountDone >= param.MaxAccounts {
		log.Info().
			Str("component", "MultiAccountSwitch").
			Int("total", multiAccountDone).
			Int("failed", multiAccountFailed).
			Int("succeeded", multiAccountDone-multiAccountFailed).
			Msg("all accounts processed")
		return false // Signal done → on_error → MultiAccountDaily_Done
	}

	ctrl := ctx.GetTasker().GetController()
	ctrl.PostClickKey(keycode.MustCode("ESC")).Wait()
	time.Sleep(1500 * time.Millisecond)
	ctrl.PostClick(350, 640).Wait()
	time.Sleep(1000 * time.Millisecond)
	ctrl.PostClick(640, 580).Wait()
	time.Sleep(2000 * time.Millisecond)

	for i := 0; i < 30; i++ {
		if ctx.GetTasker().Stopping() {
			return false
		}
		detail, err := ctx.RunRecognition(
			"__MultiAccount_LoginScreen",
			nil,
			`{
				"__MultiAccount_LoginScreen": {
					"recognition": "OCR",
					"roi": [300, 200, 680, 320]
				}
			}`,
		)
		if err == nil && detail != nil && detail.Hit {
			log.Info().Str("component", "MultiAccountSwitch").Msg("arrived at login screen")
			return true
		}
		time.Sleep(1000 * time.Millisecond)
	}

	log.Warn().Str("component", "MultiAccountSwitch").Msg("timeout waiting for login screen")
	return false
}

// verifyAccountName retries OCR verification of the selected account name up to 5 times
// (matching ok-ww's retry behavior). Returns true if the displayed name matches expected.
func (a *MultiAccountSwitchAction) verifyAccountName(ctx *maa.Context, expectedName string) bool {
	for retry := 0; retry < 5; retry++ {
		detail, err := ctx.RunRecognition(
			"__MultiAccount_VerifyName",
			nil,
			`{
				"__MultiAccount_VerifyName": {
					"recognition": "OCR",
					"roi": [400, 180, 480, 60]
				}
			}`,
		)
		if err == nil && detail != nil && detail.Hit && strings.Contains(detail.DetailJson, expectedName) {
			log.Info().Int("retry", retry).Str("expected", expectedName).Msg("account name verified")
			return true
		}
		log.Debug().Int("retry", retry).Str("expected", expectedName).Msg("account name mismatch, retrying")
		// Re-click account dropdown and re-select
		ctrl := ctx.GetTasker().GetController()
		ctrl.PostClick(600, 280).Wait()
		time.Sleep(500 * time.Millisecond)
	}
	log.Warn().Str("expected", expectedName).Msg("account name verification failed after 5 retries")
	return false
}

// ---------------------------------------------------------------------------
// MultiAccountMarkFailedAction — marks the current account as failed and skips
// to the next account.
// ---------------------------------------------------------------------------

type MultiAccountMarkFailedAction struct{}

var _ maa.CustomActionRunner = &MultiAccountMarkFailedAction{}

func (a *MultiAccountMarkFailedAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := multiAccountParam{MaxAccounts: 2}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "MultiAccountMarkFailed").Msg("failed to parse param")
		}
	}

	multiAccountFailed++
	multiAccountDone++
	log.Warn().
		Str("component", "MultiAccountMarkFailed").
		Int("failed", multiAccountFailed).
		Int("done", multiAccountDone).
		Int("max", param.MaxAccounts).
		Msg("account failed, skipping to next")

	if multiAccountDone >= param.MaxAccounts {
		log.Info().
			Int("total", multiAccountDone).
			Int("failed", multiAccountFailed).
			Int("succeeded", multiAccountDone-multiAccountFailed).
			Msg("all accounts processed (some failed)")
		return false
	}

	ctrl := ctx.GetTasker().GetController()
	ctrl.PostClickKey(keycode.MustCode("ESC")).Wait()
	time.Sleep(1500 * time.Millisecond)
	ctrl.PostClick(350, 640).Wait()
	time.Sleep(1000 * time.Millisecond)
	ctrl.PostClick(640, 580).Wait()
	time.Sleep(2000 * time.Millisecond)

	return true
}

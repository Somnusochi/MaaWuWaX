// Package navigation implements map navigation Custom Recognition for Wuthering Waves.
package navigation

import (
	"fmt"
	"strings"
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
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

func (r *MinimapNavigateRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	ctrl := ctx.GetTasker().GetController()
	walk.StepForward(ctrl)
	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"action":"step_forward"}`,
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

// ---------------------------------------------------------------------------
// ClickFastTravelAction — clicks the visible fast-travel button and handles the
// optional custom teleport confirmation.
// ---------------------------------------------------------------------------

type ClickFastTravelAction struct{}

var _ maa.CustomActionRunner = &ClickFastTravelAction{}

func (a *ClickFastTravelAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	ctrl := ctx.GetTasker().GetController()
	for _, template := range []string{
		"fast_travel_custom.png",
		"gray_teleport.png",
		"custom_teleport_hcenter_vcenter.png",
	} {
		detail, err := ctx.RunRecognition(
			"__ClickFastTravel_Button",
			nil,
			fmt.Sprintf(`{
				"__ClickFastTravel_Button": {
					"recognition": "TemplateMatch",
					"template": %q,
					"threshold": 0.7
				}
			}`, template),
		)
		if err != nil || detail == nil || !detail.Hit {
			continue
		}

		ctrl.PostClick(
			int32(detail.Box[0]+detail.Box[2]/2),
			int32(detail.Box[1]+detail.Box[3]/2),
		).Wait()
		time.Sleep(800 * time.Millisecond)

		if template != "gray_teleport.png" {
			a.confirmCustomTeleport(ctx)
		}

		log.Info().
			Str("component", "ClickFastTravel").
			Str("template", template).
			Msg("clicked fast travel")
		return true
	}

	log.Debug().Str("component", "ClickFastTravel").Msg("fast travel button not found")
	return false
}

func (a *ClickFastTravelAction) confirmCustomTeleport(ctx *maa.Context) {
	ctrl := ctx.GetTasker().GetController()
	ctrl.PostClick(627, 396).Wait()
	time.Sleep(300 * time.Millisecond)

	for _, template := range []string{
		"confirm_btn_highlight_hcenter_vcenter.png",
		"confirm_btn_hcenter_vcenter.png",
	} {
		detail, err := ctx.RunRecognition(
			"__ClickFastTravel_Confirm",
			nil,
			fmt.Sprintf(`{
				"__ClickFastTravel_Confirm": {
					"recognition": "TemplateMatch",
					"template": %q,
					"threshold": 0.6
				}
			}`, template),
		)
		if err != nil || detail == nil || !detail.Hit {
			continue
		}

		ctrl.PostClick(
			int32(detail.Box[0]+detail.Box[2]/2),
			int32(detail.Box[1]+detail.Box[3]/2),
		).Wait()
		time.Sleep(500 * time.Millisecond)
		return
	}
}

// ---------------------------------------------------------------------------
// TeleportBossAction — navigates F2 book to teleport to a boss.
// This is a simplified version; full boss selection would require OCR/index.
// ---------------------------------------------------------------------------

type TeleportBossAction struct{}

var _ maa.CustomActionRunner = &TeleportBossAction{}

type bossTeleportParam struct {
	BossType          string `json:"boss_type"`
	BossName          string `json:"boss_name"`
	BossProfile       string `json:"boss_profile"`
	SerialNumber      int    `json:"serial_number"`
	TotalNumber       int    `json:"total_number"`
	BossLevel         string `json:"boss_level"`
	WalkAfterTeleport bool   `json:"walk_after_teleport"`
	CombatWaitMs      int    `json:"combat_wait_ms"`
}

func (a *TeleportBossAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := bossTeleportParam{BossType: "boss", SerialNumber: 1, TotalNumber: 20, BossLevel: "80", WalkAfterTeleport: true}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "TeleportBoss").Msg("failed to parse param")
		}
	}
	if param.SerialNumber < 1 {
		param.SerialNumber = 1
	}
	if param.TotalNumber < param.SerialNumber {
		param.TotalNumber = param.SerialNumber
	}
	tabTemplate := "gray_book_boss.png"
	if param.BossType == "weekly" {
		tabTemplate = "book_zhange.png"
		if param.TotalNumber < 9 {
			param.TotalNumber = 9
		}
	}

	log.Info().
		Str("component", "TeleportBoss").
		Str("boss_type", param.BossType).
		Str("boss_name", param.BossName).
		Str("boss_profile", param.BossProfile).
		Str("tab_template", tabTemplate).
		Int("serial_number", param.SerialNumber).
		Int("total_number", param.TotalNumber).
		Str("boss_level", param.BossLevel).
		Msg("teleporting to boss")
	ctrl := ctx.GetTasker().GetController()

	f2Code := keycode.MustCode("F2")
	escCode := keycode.MustCode("ESC")

	// Step 1: Open F2 book.
	ctrl.PostClickKey(f2Code).Wait()
	time.Sleep(2000 * time.Millisecond)

	// Step 2: Click boss tab.
	detail, err := ctx.RunRecognition(
		"__TeleportBoss_FindTab",
		nil,
		fmt.Sprintf(`{
			"__TeleportBoss_FindTab": {
				"recognition": "TemplateMatch",
				"template": %q,
				"threshold": 0.6
			}
		}`, tabTemplate),
	)
	if err != nil || detail == nil || !detail.Hit {
		log.Warn().Str("component", "TeleportBoss").Msg("boss tab not found")
		ctrl.PostClickKey(escCode).Wait()
		return false
	}
	ctrl.PostClick(
		int32(detail.Box[0]+detail.Box[2]/2),
		int32(detail.Box[1]+detail.Box[3]/2),
	).Wait()
	time.Sleep(500 * time.Millisecond)

	if strings.TrimSpace(param.BossName) == "" || !a.selectBookTargetByName(ctx, param.BossName, param.TotalNumber) {
		a.selectBookTarget(ctx, param.SerialNumber, param.TotalNumber)
	}

	// Step 4: Click proceed button.
	proceedDetail, err := ctx.RunRecognition(
		"__TeleportBoss_Proceed",
		nil,
		`{
			"__TeleportBoss_Proceed": {
				"recognition": "TemplateMatch",
				"template": "boss_proceed.png",
				"threshold": 0.7
			}
		}`,
	)
	if err != nil || proceedDetail == nil || !proceedDetail.Hit {
		log.Warn().Str("component", "TeleportBoss").Msg("proceed button not found")
		ctrl.PostClickKey(escCode).Wait()
		return false
	}
	ctrl.PostClick(
		int32(proceedDetail.Box[0]+proceedDetail.Box[2]/2),
		int32(proceedDetail.Box[1]+proceedDetail.Box[3]/2),
	).Wait()
	time.Sleep(1000 * time.Millisecond)

	// Step 5: Click travel button.
	travelDetail, err := ctx.RunRecognition(
		"__TeleportBoss_Travel",
		nil,
		`{
			"__TeleportBoss_Travel": {
				"recognition": "TemplateMatch",
				"template": "gray_teleport.png",
				"threshold": 0.7
			}
		}`,
	)
	if err != nil || travelDetail == nil || !travelDetail.Hit {
		log.Warn().Str("component", "TeleportBoss").Msg("travel button not found")
		ctrl.PostClickKey(escCode).Wait()
		return false
	}
	ctrl.PostClick(
		int32(travelDetail.Box[0]+travelDetail.Box[2]/2),
		int32(travelDetail.Box[1]+travelDetail.Box[3]/2),
	).Wait()
	time.Sleep(3000 * time.Millisecond)

	// Step 6: Wait for world load.
	for i := 0; i < 15; i++ {
		if ctx.GetTasker().Stopping() {
			return true
		}
		ctrl.PostScreencap().Wait()
		img, err := ctrl.CacheImage()
		if err != nil {
			time.Sleep(2000 * time.Millisecond)
			continue
		}
		worldDetail, err := ctx.RunRecognition(
			"__TeleportBoss_WaitWorld",
			img,
			`{
				"__TeleportBoss_WaitWorld": {
					"recognition": "TemplateMatch",
					"template": "minimap.png",
					"threshold": 0.7,
					"roi": [1050, 20, 200, 160]
				}
			}`,
		)
		if err == nil && worldDetail != nil && worldDetail.Hit {
			log.Info().Str("component", "TeleportBoss").Msg("arrived in world")
			if param.WalkAfterTeleport {
				a.walkAfterTeleport(ctx, param)
			}
			return true
		}
		time.Sleep(2000 * time.Millisecond)
	}

	log.Warn().Str("component", "TeleportBoss").Msg("timeout waiting for world load")
	return false
}

func (a *TeleportBossAction) selectBookTarget(ctx *maa.Context, serialNumber int, totalNumber int) {
	ctrl := ctx.GetTasker().GetController()
	row := serialNumber
	if row > 4 {
		barTop := 112
		barBottom := 634
		y := barTop + (barBottom-barTop)*serialNumber/totalNumber
		ctrl.PostClick(1245, int32(y)).Wait()
		time.Sleep(900 * time.Millisecond)
		row = 4
	}
	y := int32(174 + (row-1)*118)
	ctrl.PostClick(1195, y).Wait()
	time.Sleep(1000 * time.Millisecond)
}

func (a *TeleportBossAction) selectBookTargetByName(ctx *maa.Context, bossName string, totalNumber int) bool {
	bossName = normalizeBossBookText(bossName)
	if bossName == "" {
		return false
	}

	ctrl := ctx.GetTasker().GetController()
	pages := 5
	if totalNumber <= 9 {
		pages = 3
	}

	for page := 0; page < pages; page++ {
		if page > 0 {
			y := 112 + (634-112)*page/(pages-1)
			ctrl.PostClick(1245, int32(y)).Wait()
			time.Sleep(800 * time.Millisecond)
		}

		if box, ok := a.findBookTargetName(ctx, bossName); ok {
			y := box[1] + box[3]/2
			if y < 120 {
				y = 174
			}
			ctrl.PostClick(1195, int32(y)).Wait()
			time.Sleep(1000 * time.Millisecond)
			log.Info().
				Str("component", "TeleportBoss").
				Str("boss_name", bossName).
				Int("page", page).
				Interface("box", box).
				Msg("selected boss by OCR name")
			return true
		}
	}

	log.Warn().
		Str("component", "TeleportBoss").
		Str("boss_name", bossName).
		Msg("boss name not found, falling back to serial number")
	return false
}

func (a *TeleportBossAction) findBookTargetName(ctx *maa.Context, bossName string) (maa.Rect, bool) {
	detail, err := ctx.RunRecognition(
		"__TeleportBoss_NameOCR",
		nil,
		`{
			"__TeleportBoss_NameOCR": {
				"recognition": "OCR",
				"roi": [300, 95, 920, 560]
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit || detail.Results == nil {
		return maa.Rect{}, false
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
		if strings.Contains(normalizeBossBookText(ocr.Text), bossName) {
			return ocr.Box, true
		}
	}
	return maa.Rect{}, false
}

func normalizeBossBookText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, " ", "")
	text = strings.ReplaceAll(text, "\n", "")
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ReplaceAll(text, "：", ":")
	return strings.ToLower(text)
}

func (a *TeleportBossAction) walkAfterTeleport(ctx *maa.Context, param bossTeleportParam) {
	ctrl := ctx.GetTasker().GetController()
	ctrl.PostKeyDown(keycode.MustCode("W")).Wait()
	defer ctrl.PostKeyUp(keycode.MustCode("W")).Wait()

	for i := 0; i < 40; i++ {
		if ctx.GetTasker().Stopping() {
			return
		}
		if a.inCombat(ctx) {
			log.Info().Str("component", "TeleportBoss").Msg("combat detected after boss teleport")
			a.waitForBossProfile(param)
			return
		}
		if a.hasFPrompt(ctx) {
			ctrl.PostKeyUp(keycode.MustCode("W")).Wait()
			ctrl.PostClickKey(keycode.MustCode("F")).Wait()
			time.Sleep(3000 * time.Millisecond)
			a.selectBossLevel(ctx, param.BossLevel)
			a.clickChallengeConfirm(ctx)
			a.afterRealmEnter(ctx, param)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (a *TeleportBossAction) afterRealmEnter(ctx *maa.Context, param bossTeleportParam) {
	profile := normalizeBossBookText(param.BossProfile)
	ctrl := ctx.GetTasker().GetController()

	switch profile {
	case "fallacy", "fallacyofnoreturn":
		pressFor(ctrl, keycode.MustCode("D"), 250*time.Millisecond)
		pressFor(ctrl, keycode.MustCode("W"), 700*time.Millisecond)
	case "fenrico":
		for i := 0; i < 3; i++ {
			if !a.hasFPrompt(ctx) {
				break
			}
			ctrl.PostClickKey(keycode.MustCode("F")).Wait()
			time.Sleep(1000 * time.Millisecond)
		}
		pressFor(ctrl, keycode.MustCode("W"), 1200*time.Millisecond)
	case "namelessexplorer":
		pressFor(ctrl, keycode.MustCode("W"), 2500*time.Millisecond)
	case "sentryconstruct", "lionessofglory":
		time.Sleep(5 * time.Second)
	case "hyvatia":
		time.Sleep(7 * time.Second)
	}

	a.waitForBossProfile(param)
}

func (a *TeleportBossAction) waitForBossProfile(param bossTeleportParam) {
	if param.CombatWaitMs > 0 {
		time.Sleep(time.Duration(param.CombatWaitMs) * time.Millisecond)
		return
	}
	switch normalizeBossBookText(param.BossProfile) {
	case "sentryconstruct", "lionessofglory", "fallacy", "fallacyofnoreturn":
		time.Sleep(5 * time.Second)
	case "hyvatia":
		time.Sleep(7 * time.Second)
	}
}

func (a *TeleportBossAction) inCombat(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition(
		"__TeleportBoss_Combat",
		nil,
		`{
			"__TeleportBoss_Combat": {
				"recognition": "TemplateMatch",
				"template": "has_target.png",
				"threshold": 0.7,
				"roi": [400, 200, 800, 600]
			}
		}`,
	)
	return err == nil && detail != nil && detail.Hit
}

func (a *TeleportBossAction) hasFPrompt(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition(
		"__TeleportBoss_FPrompt",
		nil,
		`{
			"__TeleportBoss_FPrompt": {
				"recognition": "Or",
				"any_of": [
					{"recognition": "TemplateMatch", "template": "pick_up_f_hcenter_vcenter.png", "threshold": 0.6},
					{"recognition": "TemplateMatch", "template": "pick_up_f.png", "threshold": 0.65},
					{"recognition": "OCR", "expected": "挑战"},
					{"recognition": "OCR", "expected": "Challenge"}
				]
			}
		}`,
	)
	return err == nil && detail != nil && detail.Hit
}

func (a *TeleportBossAction) selectBossLevel(ctx *maa.Context, level string) {
	if level == "" {
		level = "80"
	}
	detail, err := ctx.RunRecognition(
		"__TeleportBoss_Level",
		nil,
		fmt.Sprintf(`{
			"__TeleportBoss_Level": {
				"recognition": "OCR",
				"expected": %q,
				"roi": [38, 81, 376, 394]
			}
		}`, level),
	)
	if err != nil || detail == nil || !detail.Hit {
		log.Warn().Str("component", "TeleportBoss").Str("boss_level", level).Msg("boss level not found")
		return
	}
	ctx.GetTasker().GetController().PostClick(
		int32(detail.Box[0]+detail.Box[2]/2),
		int32(detail.Box[1]+detail.Box[3]/2),
	).Wait()
	time.Sleep(1000 * time.Millisecond)
}

func (a *TeleportBossAction) clickChallengeConfirm(ctx *maa.Context) {
	ctrl := ctx.GetTasker().GetController()
	for _, pos := range [][2]int32{{1126, 656}, {1162, 662}} {
		ctrl.PostClick(pos[0], pos[1]).Wait()
		time.Sleep(2000 * time.Millisecond)
	}
}

func pressFor(ctrl *maa.Controller, code int32, duration time.Duration) {
	ctrl.PostKeyDown(code).Wait()
	time.Sleep(duration)
	ctrl.PostKeyUp(code).Wait()
}

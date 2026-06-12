// Package navigation implements map navigation Custom Recognition for Wuthering Waves.
package navigation

import (
	"strings"
	"sync"
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/walk"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

var lastLoreleiNightChange time.Time
var bossBookState struct {
	sync.Mutex
	param bossTeleportParam
}

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
	for _, button := range []struct {
		node          string
		confirmCustom bool
	}{
		{node: "Navigation_FastTravelCustomButton", confirmCustom: true},
		{node: "Navigation_FastTravelGrayButton"},
		{node: "Navigation_FastTravelCenterButton", confirmCustom: true},
	} {
		if !clickRecognitionCenter(ctx, button.node) {
			continue
		}
		time.Sleep(800 * time.Millisecond)

		if button.confirmCustom {
			a.confirmCustomTeleport(ctx)
		}

		log.Info().
			Str("component", "ClickFastTravel").
			Str("node", button.node).
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

	if clickRecognitionCenter(ctx, "Navigation_FastTravelConfirmButton") {
		time.Sleep(500 * time.Millisecond)
	}
}

// ---------------------------------------------------------------------------
// TeleportBossAction — navigates the F2 book to teleport to a boss, using
// explicit OCR name matching when available and profile aliases as fallback.
// ---------------------------------------------------------------------------

type TeleportBossAction struct{}
type BossBookPrepareProfileAction struct{}
type BossBookTargetSelectAction struct{}

var _ maa.CustomActionRunner = &TeleportBossAction{}
var _ maa.CustomActionRunner = &BossBookPrepareProfileAction{}
var _ maa.CustomActionRunner = &BossBookTargetSelectAction{}

type bossTeleportParam struct {
	BossType               string `json:"boss_type"`
	BossName               string `json:"boss_name"`
	BossProfile            string `json:"boss_profile"`
	SerialNumber           int    `json:"serial_number"`
	TotalNumber            int    `json:"total_number"`
	BossLevel              string `json:"boss_level"`
	WalkAfterTeleport      bool   `json:"walk_after_teleport"`
	WaitWorldAfterTeleport bool   `json:"wait_world_after_teleport"`
	CombatWaitMs           int    `json:"combat_wait_ms"`
}

func (a *TeleportBossAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := parseBossTeleportParam(arg, true)
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

	a.prepareBossProfile(ctx, param)

	f2Code := keycode.MustCode("F2")
	escCode := keycode.MustCode("ESC")

	// Step 1: Open F2 book.
	ctrl.PostClickKey(f2Code).Wait()
	time.Sleep(2000 * time.Millisecond)

	// Step 2: Click boss tab.
	tabNode := "Navigation_BossTabBoss"
	if param.BossType == "weekly" {
		tabNode = "Navigation_BossTabWeekly"
	}
	detail, err := ctx.RunRecognition(tabNode, nil)
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

	if !a.selectBookTargetByName(ctx, param, param.TotalNumber) {
		a.selectBookTarget(ctx, param.SerialNumber, param.TotalNumber)
	}

	// Step 4: Click proceed button.
	proceedDetail, err := ctx.RunRecognition("Navigation_BossProceedButton", nil)
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
	travelDetail, err := ctx.RunRecognition("Navigation_BossTravelButton", nil)
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

	if !param.WaitWorldAfterTeleport {
		return true
	}

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
		worldDetail, err := ctx.RunRecognition("Navigation_WaitInWorld", img)
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

func (a *BossBookPrepareProfileAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := parseBossTeleportParam(arg, false)
	saveBossBookParam(param)
	(&TeleportBossAction{}).prepareBossProfile(ctx, param)
	return true
}

func (a *BossBookTargetSelectAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := loadBossBookParam()
	if arg != nil && arg.CustomActionParam != "" {
		param = parseBossTeleportParam(arg, false)
		saveBossBookParam(param)
	}
	teleport := &TeleportBossAction{}
	if !teleport.selectBookTargetByName(ctx, param, param.TotalNumber) {
		teleport.selectBookTarget(ctx, param.SerialNumber, param.TotalNumber)
	}
	return true
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
	return param
}

func saveBossBookParam(param bossTeleportParam) {
	bossBookState.Lock()
	bossBookState.param = param
	bossBookState.Unlock()
}

func loadBossBookParam() bossTeleportParam {
	bossBookState.Lock()
	defer bossBookState.Unlock()
	if bossBookState.param.SerialNumber == 0 {
		return parseBossTeleportParam(nil, false)
	}
	return bossBookState.param
}

func (a *TeleportBossAction) prepareBossProfile(ctx *maa.Context, param bossTeleportParam) {
	switch normalizeBossBookText(param.BossProfile) {
	case "lorelei":
		a.ensureNightForLorelei(ctx)
	}
}

func (a *TeleportBossAction) ensureNightForLorelei(ctx *maa.Context) {
	elapsed := time.Since(lastLoreleiNightChange)
	if !lastLoreleiNightChange.IsZero() && elapsed <= 11*time.Minute {
		log.Info().
			Str("component", "TeleportBoss").
			Dur("elapsed", elapsed).
			Msg("skip lorelei night change; recent enough")
		return
	}

	log.Info().Str("component", "TeleportBoss").Msg("changing time to night for Lorelei")
	ctrl := ctx.GetTasker().GetController()
	escCode := keycode.MustCode("ESC")

	ctrl.PostClickKey(escCode).Wait()
	time.Sleep(1 * time.Second)
	ctrl.PostClick(909, 691).Wait()
	time.Sleep(2 * time.Second)
	ctrl.PostClick(243, 101).Wait()
	time.Sleep(1 * time.Second)

	for i := 0; i < 3; i++ {
		ctrl.PostClick(1050, 382).Wait()
		time.Sleep(1 * time.Second)
	}

	ctrl.PostClick(666, 648).Wait()
	time.Sleep(6 * time.Second)
	ctrl.PostClickKey(escCode).Wait()
	time.Sleep(1 * time.Second)
	ctrl.PostClickKey(escCode).Wait()
	time.Sleep(1 * time.Second)

	lastLoreleiNightChange = time.Now()
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

func (a *TeleportBossAction) selectBookTargetByName(ctx *maa.Context, param bossTeleportParam, totalNumber int) bool {
	queries := bossBookQueries(param)
	if len(queries) == 0 {
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

		if query, box, ok := a.findBookTargetName(ctx, queries); ok {
			y := box[1] + box[3]/2
			if y < 120 {
				y = 174
			}
			ctrl.PostClick(1195, int32(y)).Wait()
			time.Sleep(1000 * time.Millisecond)
			log.Info().
				Str("component", "TeleportBoss").
				Str("boss_name", query).
				Str("boss_profile", param.BossProfile).
				Int("page", page).
				Interface("box", box).
				Msg("selected boss by OCR name")
			return true
		}
	}

	log.Warn().
		Str("component", "TeleportBoss").
		Strs("boss_queries", queries).
		Str("boss_profile", param.BossProfile).
		Msg("boss name/profile not found, falling back to serial number")
	return false
}

func (a *TeleportBossAction) findBookTargetName(ctx *maa.Context, queries []string) (string, maa.Rect, bool) {
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
	default:
		return nil
	}
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
		wait := time.Duration(param.CombatWaitMs) * time.Millisecond
		if param.CombatWaitMs <= 120 {
			wait = time.Duration(param.CombatWaitMs) * time.Second
		}
		time.Sleep(wait)
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
	detail, err := ctx.RunRecognition("Navigation_CombatTarget", nil)
	return err == nil && detail != nil && detail.Hit
}

func (a *TeleportBossAction) hasFPrompt(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition("Navigation_FPrompt", nil)
	return err == nil && detail != nil && detail.Hit
}

func (a *TeleportBossAction) selectBossLevel(ctx *maa.Context, level string) {
	if level == "" {
		level = "80"
	}
	detail, err := ctx.RunRecognition("Navigation_BossLevelOCR", nil)
	if err != nil || detail == nil || !detail.Hit || detail.Results == nil {
		log.Warn().Str("component", "TeleportBoss").Str("boss_level", level).Msg("boss level not found")
		return
	}
	box, ok := findOCRTextBox(detail, level)
	if !ok {
		log.Warn().Str("component", "TeleportBoss").Str("boss_level", level).Msg("boss level not found")
		return
	}
	ctx.GetTasker().GetController().PostClick(
		int32(box[0]+box[2]/2),
		int32(box[1]+box[3]/2),
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

func findOCRTextBox(detail *maa.RecognitionDetail, expected string) (maa.Rect, bool) {
	expected = normalizeBossBookText(expected)
	results := detail.Results.Filtered
	if len(results) == 0 {
		results = detail.Results.All
	}
	for _, result := range results {
		ocr, ok := result.AsOCR()
		if !ok || ocr == nil {
			continue
		}
		if strings.Contains(normalizeBossBookText(ocr.Text), expected) {
			return ocr.Box, true
		}
	}
	return maa.Rect{}, false
}

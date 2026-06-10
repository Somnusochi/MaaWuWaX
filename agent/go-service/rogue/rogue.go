// Package rogue implements half-auto rogue Custom Actions for Wuthering Waves.
package rogue

import (
	"fmt"
	"strings"
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
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

type rogueBuffChoice struct {
	Text string
	Box  maa.Rect
}

type rogueBuffOCRPiece struct {
	Text string
	Box  maa.Rect
}

func defaultRogueBuffParam() rogueBuffParam {
	return rogueBuffParam{
		Blacklist: []string{"雷暴", "旋风", "矛盾晶体"},
		Whitelist: []string{"心流", "悲鸣纪", "余音贝", "齿轮之心", "全知之眼", "指南针", "医疗箱"},
	}
}

// parseRogueBuffParamFromCSV handles input-type options that pass
// comma-separated values instead of JSON arrays.
func parseRogueBuffParamFromCSV(raw string) rogueBuffParam {
	// Try to extract whitelist/blacklist from a flat JSON string like
	// {"whitelist":"a,b,c","blacklist":"d,e"} or just the raw CSV text.
	type flatParam struct {
		Whitelist string `json:"whitelist"`
		Blacklist string `json:"blacklist"`
	}
	var flat flatParam
	if err := sonic.Unmarshal([]byte(raw), &flat); err == nil {
		return rogueBuffParam{
			Whitelist: splitCSV(flat.Whitelist),
			Blacklist: splitCSV(flat.Blacklist),
		}
	}
	// Fallback: treat entire raw string as whitelist if non-empty.
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" && !strings.HasPrefix(trimmed, "{") {
		return rogueBuffParam{
			Whitelist: splitCSV(trimmed),
			Blacklist: defaultRogueBuffParam().Blacklist,
		}
	}
	return rogueBuffParam{}
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func (a *RogueBuffSelectAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := defaultRogueBuffParam()
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			// Try parsing as comma-separated strings (from input-type options)
			param = parseRogueBuffParamFromCSV(arg.CustomActionParam)
			if len(param.Whitelist) == 0 && len(param.Blacklist) == 0 {
				log.Warn().Err(err).Str("component", "RogueBuffSelect").Msg("failed to parse param")
				param = defaultRogueBuffParam()
			}
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

	choices := rogueBuffChoices(detail)
	text := detail.DetailJson
	// Try whitelist first.
	for _, w := range param.Whitelist {
		if choice, ok := findRogueBuffChoice(choices, w); ok {
			clickRogueBuffChoice(ctrl, choice)
			log.Info().
				Str("component", "RogueBuffSelect").
				Str("buff", w).
				Str("text", choice.Text).
				Interface("box", choice.Box).
				Msg("selected whitelist buff")
			time.Sleep(1000 * time.Millisecond)
			return true
		}
		if strings.Contains(text, w) {
			log.Info().Str("component", "RogueBuffSelect").Str("buff", w).Msg("selected whitelist buff by fallback text")
			clickFallbackBuff(ctrl, choices)
			time.Sleep(1000 * time.Millisecond)
			return true
		}
	}

	// No whitelist hit: choose the first OCR choice that is not blacklisted.
	for _, choice := range choices {
		if !rogueBuffChoiceBlacklisted(choice, param.Blacklist) {
			clickRogueBuffChoice(ctrl, choice)
			log.Info().
				Str("component", "RogueBuffSelect").
				Str("text", choice.Text).
				Interface("box", choice.Box).
				Msg("selected non-blacklisted buff")
			time.Sleep(1000 * time.Millisecond)
			return true
		}
	}

	clickFallbackBuff(ctrl, choices)
	time.Sleep(1000 * time.Millisecond)
	return true
}

func rogueBuffChoices(detail *maa.RecognitionDetail) []rogueBuffChoice {
	pieces := rogueBuffOCRPieces(detail)
	if len(pieces) == 0 {
		return nil
	}
	grouped := groupRogueBuffCards(pieces)
	if len(grouped) > 0 {
		return grouped
	}
	choices := make([]rogueBuffChoice, 0, len(pieces))
	for _, piece := range pieces {
		choices = append(choices, rogueBuffChoice{Text: piece.Text, Box: piece.Box})
	}
	return choices
}

func rogueBuffOCRPieces(detail *maa.RecognitionDetail) []rogueBuffOCRPiece {
	if detail == nil || detail.Results == nil {
		return nil
	}
	results := detail.Results.Filtered
	if len(results) == 0 {
		results = detail.Results.All
	}
	choices := make([]rogueBuffOCRPiece, 0, len(results))
	for _, result := range results {
		ocr, ok := result.AsOCR()
		if !ok || ocr == nil {
			continue
		}
		text := normalizeRogueBuffText(ocr.Text)
		if text == "" {
			continue
		}
		choices = append(choices, rogueBuffOCRPiece{Text: text, Box: ocr.Box})
	}
	return choices
}

func groupRogueBuffCards(pieces []rogueBuffOCRPiece) []rogueBuffChoice {
	type agg struct {
		texts []string
		box   maa.Rect
		count int
	}
	var buckets [3]agg
	for _, piece := range pieces {
		cx := piece.Box[0] + piece.Box[2]/2
		idx := 1
		switch {
		case cx < 507:
			idx = 0
		case cx > 773:
			idx = 2
		}
		bucket := &buckets[idx]
		if bucket.count == 0 {
			bucket.box = piece.Box
		} else {
			bucket.box = mergeRogueRects(bucket.box, piece.Box)
		}
		bucket.texts = append(bucket.texts, piece.Text)
		bucket.count++
	}

	choices := make([]rogueBuffChoice, 0, 3)
	for _, bucket := range buckets {
		if bucket.count == 0 {
			continue
		}
		text := normalizeRogueBuffText(strings.Join(bucket.texts, ""))
		if text == "" {
			continue
		}
		choices = append(choices, rogueBuffChoice{
			Text: text,
			Box:  bucket.box,
		})
	}
	return choices
}

func mergeRogueRects(a, b maa.Rect) maa.Rect {
	left := minInt(a[0], b[0])
	top := minInt(a[1], b[1])
	right := maxInt(a[0]+a[2], b[0]+b[2])
	bottom := maxInt(a[1]+a[3], b[1]+b[3])
	return maa.Rect{left, top, right - left, bottom - top}
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

func normalizeRogueBuffText(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, " ", ""))
}

func findRogueBuffChoice(choices []rogueBuffChoice, target string) (rogueBuffChoice, bool) {
	target = normalizeRogueBuffText(target)
	for _, choice := range choices {
		if strings.Contains(choice.Text, target) {
			return choice, true
		}
	}
	return rogueBuffChoice{}, false
}

func rogueBuffChoiceBlacklisted(choice rogueBuffChoice, blacklist []string) bool {
	for _, blocked := range blacklist {
		if strings.Contains(choice.Text, normalizeRogueBuffText(blocked)) {
			return true
		}
	}
	return false
}

func clickRogueBuffChoice(ctrl *maa.Controller, choice rogueBuffChoice) {
	x := choice.Box[0] + choice.Box[2]/2
	if x < 373 {
		x = 373
	} else if x > 907 {
		x = 907
	}
	y := choice.Box[1] + choice.Box[3]/2
	if y < 395 {
		y = 395
	} else if y > 455 {
		y = 455
	}
	ctrl.PostClick(int32(x), int32(y)).Wait()
}

func clickFallbackBuff(ctrl *maa.Controller, choices []rogueBuffChoice) {
	if len(choices) > 0 {
		clickRogueBuffChoice(ctrl, choices[0])
		return
	}
	// Absolute last resort: click middle of buff list area.
	ctrl.PostClick(640, 430).Wait()
	log.Warn().Str("component", "RogueBuffSelect").Msg("fallback to absolute middle click")
}

type RogueTreasureRewardAction struct {
	claimed int
}

var _ maa.CustomActionRunner = &RogueTreasureRewardAction{}

type rogueTreasureParam struct {
	StopOnTreasure bool `json:"stop_on_treasure"`
	MaxClaims      int  `json:"max_claims"`
}

func (a *RogueTreasureRewardAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := rogueTreasureParam{MaxClaims: 2}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "RogueTreasureReward").Msg("failed to parse param")
		}
	}
	if param.StopOnTreasure {
		log.Info().Str("component", "RogueTreasureReward").Msg("treasure found, stopping by option")
		ctx.GetTasker().PostStop().Wait()
		return true
	}
	if param.MaxClaims <= 0 {
		param.MaxClaims = 2
	}

	ctrl := ctx.GetTasker().GetController()
	if a.hasStaminaRefill(ctx) || a.claimed >= param.MaxClaims {
		ctrl.PostClickKey(keycode.MustCode("ESC")).Wait()
		time.Sleep(1000 * time.Millisecond)
		log.Info().Str("component", "RogueTreasureReward").Int("claimed", a.claimed).Msg("skip treasure reward")
		return true
	}

	if a.claimed == 0 {
		ctrl.PostClick(870, 454).Wait()
	} else {
		ctrl.PostClick(410, 446).Wait()
	}
	a.claimed++
	time.Sleep(1500 * time.Millisecond)
	log.Info().Str("component", "RogueTreasureReward").Int("claimed", a.claimed).Msg("claimed treasure reward")
	return true
}

func (a *RogueTreasureRewardAction) hasStaminaRefill(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition(
		"__RogueTreasure_StaminaRefill",
		nil,
		`{
			"__RogueTreasure_StaminaRefill": {
				"recognition": "Or",
				"any_of": [
					{"recognition": "OCR", "expected": "补充结晶"},
					{"recognition": "OCR", "expected": "Refill"}
				]
			}
		}`,
	)
	return err == nil && detail != nil && detail.Hit
}

type RogueWalkToTargetAction struct{}

var _ maa.CustomActionRunner = &RogueWalkToTargetAction{}

type RogueWalkGateAction struct{}

var _ maa.CustomActionRunner = &RogueWalkGateAction{}

type rogueWalkParam struct {
	Template   string `json:"template"`
	DurationMs int    `json:"duration_ms"`
}

func (a *RogueWalkToTargetAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := rogueWalkParam{Template: "purple_target_distance_icon.png", DurationMs: 900}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "RogueWalkToTarget").Msg("failed to parse param")
		}
	}

	detail, err := ctx.RunRecognition(
		"__RogueWalk_Target",
		nil,
		fmt.Sprintf(`{
			"__RogueWalk_Target": {
				"recognition": "TemplateMatch",
				"template": %q,
				"threshold": 0.6,
				"roi": [230, 72, 820, 510]
			}
		}`, param.Template),
	)
	ctrl := ctx.GetTasker().GetController()
	if err != nil || detail == nil || !detail.Hit {
		pressFor(ctrl, keycode.MustCode("D"), 250*time.Millisecond)
		pressFor(ctrl, keycode.MustCode("W"), time.Duration(param.DurationMs)*time.Millisecond)
		return true
	}

	centerX := detail.Box[0] + detail.Box[2]/2
	duration := time.Duration(param.DurationMs) * time.Millisecond
	if centerX < 540 {
		keyDownPair(ctrl, keycode.MustCode("A"), keycode.MustCode("W"))
		time.Sleep(duration / 2)
		keyUpPair(ctrl, keycode.MustCode("A"), keycode.MustCode("W"))
	} else if centerX > 740 {
		keyDownPair(ctrl, keycode.MustCode("D"), keycode.MustCode("W"))
		time.Sleep(duration / 2)
		keyUpPair(ctrl, keycode.MustCode("D"), keycode.MustCode("W"))
	} else {
		pressFor(ctrl, keycode.MustCode("W"), duration)
	}

	log.Debug().
		Str("component", "RogueWalkToTarget").
		Str("template", param.Template).
		Int("center_x", centerX).
		Msg("walked toward target")
	return true
}

func (a *RogueWalkGateAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	ctrl := ctx.GetTasker().GetController()
	for i := 0; i < 4; i++ {
		if hasRogueFPrompt(ctx) {
			ctrl.PostClickKey(keycode.MustCode("F")).Wait()
			time.Sleep(1000 * time.Millisecond)
			return true
		}

		box, ok := findRogueGate(ctx)
		if ok {
			centerX := box[0] + box[2]/2
			if centerX < 448 {
				keyDownPair(ctrl, keycode.MustCode("A"), keycode.MustCode("W"))
				time.Sleep(250 * time.Millisecond)
				keyUpPair(ctrl, keycode.MustCode("A"), keycode.MustCode("W"))
			} else if centerX > 832 {
				keyDownPair(ctrl, keycode.MustCode("D"), keycode.MustCode("W"))
				time.Sleep(250 * time.Millisecond)
				keyUpPair(ctrl, keycode.MustCode("D"), keycode.MustCode("W"))
			}
			pressFor(ctrl, keycode.MustCode("W"), 1200*time.Millisecond)
			log.Info().
				Str("component", "RogueWalkGate").
				Int("center_x", centerX).
				Interface("box", box).
				Msg("walked toward gate")
			return true
		}

		pressFor(ctrl, keycode.MustCode("D"), 300*time.Millisecond)
		pressFor(ctrl, keycode.MustCode("W"), 500*time.Millisecond)
		time.Sleep(300 * time.Millisecond)
	}

	pressFor(ctrl, keycode.MustCode("W"), 2000*time.Millisecond)
	return true
}

func findRogueGate(ctx *maa.Context) (maa.Rect, bool) {
	detail, err := ctx.RunRecognition(
		"__RogueGate_OCR",
		nil,
		`{
			"__RogueGate_OCR": {
				"recognition": "OCR",
				"roi": [0, 0, 1280, 720]
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
		text := normalizeRogueGateText(ocr.Text)
		if strings.Contains(text, "的记忆") ||
			strings.Contains(text, "梦乡的") ||
			strings.Contains(text, "记忆区") ||
			strings.Contains(text, "前往下一") {
			return ocr.Box, true
		}
	}
	return maa.Rect{}, false
}

func normalizeRogueGateText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, " ", "")
	text = strings.ReplaceAll(text, "\n", "")
	return strings.ToLower(text)
}

func hasRogueFPrompt(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition(
		"__RogueGate_F",
		nil,
		`{
			"__RogueGate_F": {
				"recognition": "Or",
				"any_of": [
					{"recognition": "TemplateMatch", "template": "pick_up_f.png", "threshold": 0.6},
					{"recognition": "TemplateMatch", "template": "pick_up_f_hcenter_vcenter.png", "threshold": 0.6}
				]
			}
		}`,
	)
	return err == nil && detail != nil && detail.Hit
}

func pressFor(ctrl *maa.Controller, code int32, duration time.Duration) {
	ctrl.PostKeyDown(code).Wait()
	time.Sleep(duration)
	ctrl.PostKeyUp(code).Wait()
}

func keyDownPair(ctrl *maa.Controller, first int32, second int32) {
	ctrl.PostKeyDown(first).Wait()
	ctrl.PostKeyDown(second).Wait()
}

func keyUpPair(ctrl *maa.Controller, first int32, second int32) {
	ctrl.PostKeyUp(second).Wait()
	ctrl.PostKeyUp(first).Wait()
}

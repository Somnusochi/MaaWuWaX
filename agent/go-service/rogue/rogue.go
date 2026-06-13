// Package rogue implements half-auto rogue Custom Actions for Wuthering Waves.
package rogue

import (
	"strings"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// RogueBuffSelectAction — OCRs buff names and selects based on whitelist/blacklist.
// ---------------------------------------------------------------------------

type RogueBuffSelectAction struct{}

var _ maa.CustomActionRunner = &RogueBuffSelectAction{}

type rogueBuffParam struct {
	Blacklist        []string    `json:"blacklist"`
	Whitelist        []string    `json:"whitelist"`
	OCRNode          string      `json:"ocr_node"`
	FallbackIndex    int         `json:"fallback_index"`
	FallbackTarget   [2]int32    `json:"fallback_target"`
	ClickBounds      rogueBounds `json:"click_bounds"`
	ColumnBoundaries [2]int      `json:"column_boundaries"`
}

type rogueBounds struct {
	MinX int `json:"min_x"`
	MaxX int `json:"max_x"`
	MinY int `json:"min_y"`
	MaxY int `json:"max_y"`
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
		Blacklist:        []string{"雷暴", "旋风", "矛盾晶体"},
		Whitelist:        []string{"心流", "悲鸣纪", "余音贝", "齿轮之心", "全知之眼", "指南针", "医疗箱", "妄语的残谱", "激越的残谱"},
		OCRNode:          "RogueBuff_OCR",
		FallbackIndex:    0,
		FallbackTarget:   [2]int32{640, 430},
		ColumnBoundaries: [2]int{507, 773},
		ClickBounds: rogueBounds{
			MinX: 373,
			MaxX: 907,
			MinY: 395,
			MaxY: 455,
		},
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
	normalizeRogueBuffParam(&param)

	ctrl := ctx.GetTasker().GetController()

	// OCR the buff area (3 buff choices in a row).
	detail, err := ctx.RunRecognition(param.OCRNode, nil)
	if err != nil || detail == nil || !detail.Hit {
		// Fallback: click middle buff.
		log.Warn().Str("component", "RogueBuffSelect").Msg("OCR failed, clicking middle")
		clickFallbackBuff(ctrl, nil, param)
		return true
	}

	choices := rogueBuffChoices(detail, param)
	text := detail.DetailJson
	// Try whitelist first.
	for _, w := range param.Whitelist {
		if choice, ok := findRogueBuffChoice(choices, w); ok {
			clickRogueBuffChoice(ctrl, choice, param)
			log.Info().
				Str("component", "RogueBuffSelect").
				Str("buff", w).
				Str("text", choice.Text).
				Interface("box", choice.Box).
				Msg("selected whitelist buff")
			return true
		}
		if strings.Contains(text, w) {
			log.Info().Str("component", "RogueBuffSelect").Str("buff", w).Msg("selected whitelist buff by fallback text")
			clickFallbackBuff(ctrl, choices, param)
			return true
		}
	}

	// No whitelist hit: choose the first OCR choice that is not blacklisted.
	for _, choice := range choices {
		if !rogueBuffChoiceBlacklisted(choice, param.Blacklist) {
			clickRogueBuffChoice(ctrl, choice, param)
			log.Info().
				Str("component", "RogueBuffSelect").
				Str("text", choice.Text).
				Interface("box", choice.Box).
				Msg("selected non-blacklisted buff")
			return true
		}
	}

	clickFallbackBuff(ctrl, choices, param)
	return true
}

func normalizeRogueBuffParam(param *rogueBuffParam) {
	defaults := defaultRogueBuffParam()
	if param.OCRNode == "" {
		param.OCRNode = defaults.OCRNode
	}
	if param.FallbackTarget == [2]int32{} {
		param.FallbackTarget = defaults.FallbackTarget
	}
	if param.ColumnBoundaries == [2]int{} {
		param.ColumnBoundaries = defaults.ColumnBoundaries
	}
	if param.ClickBounds == (rogueBounds{}) {
		param.ClickBounds = defaults.ClickBounds
	}
	if len(param.Whitelist) == 0 {
		param.Whitelist = defaults.Whitelist
	}
	if len(param.Blacklist) == 0 {
		param.Blacklist = defaults.Blacklist
	}
}

func rogueBuffChoices(detail *maa.RecognitionDetail, param rogueBuffParam) []rogueBuffChoice {
	pieces := rogueBuffOCRPieces(detail)
	if len(pieces) == 0 {
		return nil
	}
	grouped := groupRogueBuffCards(pieces, param.ColumnBoundaries)
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

func groupRogueBuffCards(pieces []rogueBuffOCRPiece, boundaries [2]int) []rogueBuffChoice {
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
		case cx < boundaries[0]:
			idx = 0
		case cx > boundaries[1]:
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

func clickRogueBuffChoice(ctrl *maa.Controller, choice rogueBuffChoice, param rogueBuffParam) {
	x := choice.Box[0] + choice.Box[2]/2
	if x < param.ClickBounds.MinX {
		x = param.ClickBounds.MinX
	} else if x > param.ClickBounds.MaxX {
		x = param.ClickBounds.MaxX
	}
	y := choice.Box[1] + choice.Box[3]/2
	if y < param.ClickBounds.MinY {
		y = param.ClickBounds.MinY
	} else if y > param.ClickBounds.MaxY {
		y = param.ClickBounds.MaxY
	}
	ctrl.PostClick(int32(x), int32(y)).Wait()
}

func clickFallbackBuff(ctrl *maa.Controller, choices []rogueBuffChoice, param rogueBuffParam) {
	if len(choices) > 0 {
		index := param.FallbackIndex
		if index < 0 {
			index = 0
		}
		if index >= len(choices) {
			index = len(choices) - 1
		}
		clickRogueBuffChoice(ctrl, choices[index], param)
		return
	}
	// Absolute last resort: click middle of buff list area.
	ctrl.PostClick(param.FallbackTarget[0], param.FallbackTarget[1]).Wait()
	log.Warn().Str("component", "RogueBuffSelect").Msg("fallback to absolute middle click")
}

type RogueGatePositionRecognition struct{}

var _ maa.CustomRecognitionRunner = &RogueGatePositionRecognition{}

type rogueGatePositionParam struct {
	OCRNode    string   `json:"ocr_node"`
	Queries    []string `json:"queries"`
	MinCenterX int      `json:"min_center_x"`
	MaxCenterX int      `json:"max_center_x"`
}

func (r *RogueGatePositionRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := rogueGatePositionParam{
		OCRNode: "RogueGate_OCR",
		Queries: []string{"的记忆", "梦乡的", "记忆区", "前往下一", "奇异的白猫"},
	}
	if arg != nil && arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "RogueGatePosition").Msg("failed to parse param")
		}
	}
	if param.OCRNode == "" {
		param.OCRNode = "RogueGate_OCR"
	}
	if len(param.Queries) == 0 {
		param.Queries = []string{"的记忆", "梦乡的", "记忆区", "前往下一", "奇异的白猫"}
	}

	box, ok := findRogueGate(ctx, param.OCRNode, param.Queries)
	if !ok {
		return nil, false
	}

	centerX := int(box[0] + box[2]/2)
	if param.MinCenterX > 0 && centerX < param.MinCenterX {
		return nil, false
	}
	if param.MaxCenterX > 0 && centerX > param.MaxCenterX {
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    box,
		Detail: `{"gate_visible":true}`,
	}, true
}

func findRogueGate(ctx *maa.Context, node string, queries []string) (maa.Rect, bool) {
	detail, err := ctx.RunRecognition(node, nil)
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
		for _, query := range queries {
			if query != "" && strings.Contains(text, normalizeRogueGateText(query)) {
				return ocr.Box, true
			}
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

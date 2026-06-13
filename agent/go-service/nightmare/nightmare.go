// Package nightmare implements Custom atoms for Nightmare Nest farming.
package nightmare

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// FindNestRecognition — OCR scans the current book page for incomplete nests.
// Matches "x/y" format where y ∈ {24, 36, 48} and x < y.
// ---------------------------------------------------------------------------

type FindNestRecognition struct{}

var _ maa.CustomRecognitionRunner = &FindNestRecognition{}

type findNestParam struct {
	Denominators []int  `json:"denominators"`
	TargetX      int    `json:"target_x"`
	MinY         int    `json:"min_y"`
	FallbackY    int    `json:"fallback_y"`
	OCRNode      string `json:"ocr_node"`
}

var countRe = regexp.MustCompile(`(\d{1,2})\s*/\s*(\d{1,2})`)

const (
	nightmareCountOCRNode     = "NightmareNest_CountOCR"
	nightmareScrollOCRNode    = "NightmareNest_ScrollOCR"
	nightmareScrollEndOCRNode = "NightmareNest_ScrollEndOCR"
	nightmareApproachCombat   = "NightmareNest_ApproachCombat"
	nightmareApproachFPrompt  = "NightmareNest_ApproachFPrompt"
)

var nightmareScrollState struct {
	sync.Mutex
	fingerprint string
}

func (r *FindNestRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := normalizeFindNestParam(findNestParam{})
	if arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "NightmareFindNest").Msg("failed to parse param")
		}
	}
	param = normalizeFindNestParam(param)

	detail, err := ctx.RunRecognition(param.OCRNode, arg.Img)
	if err != nil || detail == nil || !detail.Hit {
		return nil, false
	}

	result, ok := findIncompleteNestResult(detail, param.Denominators, param.TargetX, param.MinY, param.FallbackY)
	if ok {
		log.Info().Str("component", "NightmareFindNest").Msg("found incomplete nest")
		return result, true
	}

	log.Debug().Str("component", "NightmareFindNest").Msg("no incomplete nest on current page")
	return nil, false
}

type nightmareOCRItem struct {
	text string
	box  maa.Rect
}

func nightmareOCRItems(detail *maa.RecognitionDetail) []nightmareOCRItem {
	if detail.Results == nil {
		return []nightmareOCRItem{{text: detail.DetailJson, box: detail.Box}}
	}
	results := detail.Results.Filtered
	if len(results) == 0 {
		results = detail.Results.All
	}
	items := make([]nightmareOCRItem, 0, len(results))
	for _, result := range results {
		ocr, ok := result.AsOCR()
		if !ok || ocr == nil {
			continue
		}
		items = append(items, nightmareOCRItem{text: ocr.Text, box: ocr.Box})
	}
	if len(items) == 0 {
		items = append(items, nightmareOCRItem{text: detail.DetailJson, box: detail.Box})
	}
	return items
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func normalizeFindNestParam(param findNestParam) findNestParam {
	if len(param.Denominators) == 0 {
		param.Denominators = []int{24, 36, 48}
	}
	if param.TargetX == 0 {
		param.TargetX = 1152
	}
	if param.MinY == 0 {
		param.MinY = 110
	}
	if param.FallbackY == 0 {
		param.FallbackY = 180
	}
	if param.OCRNode == "" {
		param.OCRNode = nightmareCountOCRNode
	}
	return param
}

type nightmareScrollFingerprintParam struct {
	Node string `json:"node"`
}

type SaveNightmareScrollFingerprintAction struct{}
type NightmareScrollFingerprintAdvancedRecognition struct{}

var _ maa.CustomActionRunner = &SaveNightmareScrollFingerprintAction{}
var _ maa.CustomRecognitionRunner = &NightmareScrollFingerprintAdvancedRecognition{}

func (a *SaveNightmareScrollFingerprintAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := parseNightmareScrollFingerprintParam(arg)
	fingerprint := nestOCRFingerprint(ctx, param.Node)

	nightmareScrollState.Lock()
	nightmareScrollState.fingerprint = fingerprint
	nightmareScrollState.Unlock()

	log.Debug().
		Str("component", "NightmareScrollFingerprintSave").
		Str("node", param.Node).
		Bool("has_fingerprint", fingerprint != "").
		Msg("saved scroll fingerprint")
	return true
}

func (r *NightmareScrollFingerprintAdvancedRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := parseNightmareScrollFingerprintParamFromRecognition(arg)
	current := nestOCRFingerprint(ctx, param.Node)

	nightmareScrollState.Lock()
	last := nightmareScrollState.fingerprint
	nightmareScrollState.Unlock()

	if current != "" && current == last {
		log.Info().
			Str("component", "NightmareScrollFingerprintAdvanced").
			Str("node", param.Node).
			Msg("scroll fingerprint unchanged")
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"scroll_advanced":true}`,
	}, true
}

func parseNightmareScrollFingerprintParam(arg *maa.CustomActionArg) nightmareScrollFingerprintParam {
	param := nightmareScrollFingerprintParam{Node: nightmareScrollEndOCRNode}
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "NightmareScrollFingerprintSave").Msg("failed to parse param")
		}
	}
	if param.Node == "" {
		param.Node = nightmareScrollEndOCRNode
	}
	return param
}

func parseNightmareScrollFingerprintParamFromRecognition(arg *maa.CustomRecognitionArg) nightmareScrollFingerprintParam {
	param := nightmareScrollFingerprintParam{Node: nightmareScrollEndOCRNode}
	if arg != nil && arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "NightmareScrollFingerprintAdvanced").Msg("failed to parse param")
		}
	}
	if param.Node == "" {
		param.Node = nightmareScrollEndOCRNode
	}
	return param
}

func findIncompleteNestResult(detail *maa.RecognitionDetail, denominators []int, targetX int, minY int, fallbackY int) (*maa.CustomRecognitionResult, bool) {
	for _, item := range nightmareOCRItems(detail) {
		text := strings.Trim(item.text, `"`)
		matches := countRe.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			current, err1 := strconv.Atoi(match[1])
			total, err2 := strconv.Atoi(match[2])
			if err1 != nil || err2 != nil || current >= total || !containsInt(denominators, total) {
				continue
			}

			y := item.box[1] + item.box[3]/2
			if y < minY {
				y = fallbackY
			}
			box := maa.Rect{targetX, y - 1, 2, 2}
			return &maa.CustomRecognitionResult{
				Box:    box,
				Detail: fmt.Sprintf(`{"current":%d,"total":%d}`, current, total),
			}, true
		}
	}

	return nil, false
}

// nestOCRFingerprint returns a stable page fingerprint for end-of-list detection.
func nestOCRFingerprint(ctx *maa.Context, node string) string {
	detail, err := ctx.RunRecognition(node, nil)
	if err != nil || detail == nil || !detail.Hit {
		return ""
	}
	items := nightmareOCRItems(detail)
	if len(items) == 0 {
		return strings.TrimSpace(detail.DetailJson)
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		text := normalizeNightmareOCR(item.text)
		if text == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s@%d", text, item.box[1]/10))
	}
	return strings.Join(parts, "|")
}

func normalizeNightmareOCR(text string) string {
	text = strings.TrimSpace(strings.Trim(text, `"`))
	text = strings.ReplaceAll(text, " ", "")
	text = strings.ReplaceAll(text, "\n", "")
	return text
}

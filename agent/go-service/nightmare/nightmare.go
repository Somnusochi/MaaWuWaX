// Package nightmare implements Custom atoms for Nightmare Nest farming.
package nightmare

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	Denominators []int `json:"denominators"`
}

var countRe = regexp.MustCompile(`(\d{1,2})\s*/\s*(\d{1,2})`)

func (r *FindNestRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := findNestParam{Denominators: []int{24, 36, 48}}
	if arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "NightmareFindNest").Msg("failed to parse param")
		}
	}

	detail, err := ctx.RunRecognition(
		"__NightmareNest_CountOCR",
		arg.Img,
		`{
			"__NightmareNest_CountOCR": {
				"recognition": "OCR",
				"roi": [460, 95, 780, 560]
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		return nil, false
	}

	for _, item := range nightmareOCRItems(detail) {
		text := strings.Trim(item.text, `"`)
		matches := countRe.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			current, err1 := strconv.Atoi(match[1])
			total, err2 := strconv.Atoi(match[2])
			if err1 != nil || err2 != nil || current >= total || !containsInt(param.Denominators, total) {
				continue
			}

			y := item.box[1] + item.box[3]/2
			if y < 110 {
				y = 180
			}
			box := maa.Rect{1152, y - 1, 2, 2}
			log.Info().
				Str("component", "NightmareFindNest").
				Int("current", current).
				Int("total", total).
				Msg("found incomplete nest")
			return &maa.CustomRecognitionResult{
				Box:    box,
				Detail: fmt.Sprintf(`{"current":%d,"total":%d}`, current, total),
			}, true
		}
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

// ---------------------------------------------------------------------------
// NestScrollRecognition — scrolls through the book page by page, OCR-scanning
// for incomplete nests on each page. Returns the nest coordinates if found.
// ---------------------------------------------------------------------------

type NestScrollRecognition struct{}

var _ maa.CustomRecognitionRunner = &NestScrollRecognition{}

type nestScrollParam struct {
	MaxPages int `json:"max_pages"`
}

func (r *NestScrollRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := nestScrollParam{MaxPages: 5}
	if arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "NestScroll").Msg("failed to parse param")
		}
	}

	// First, scan the current page without scrolling.
	if result, ok := nestScanPage(ctx); ok {
		return result, true
	}

	ctrl := ctx.GetTasker().GetController()
	var lastOCRText string

	for page := 1; page <= param.MaxPages; page++ {
		if ctx.GetTasker().Stopping() {
			return nil, false
		}

		log.Debug().Int("page", page).Str("component", "NestScroll").Msg("scrolling book")

		// Scroll down by clicking on the scroll bar at a position proportional to page.
		barTop := 112
		barBottom := 634
		y := barTop + (barBottom-barTop)*page/(param.MaxPages+1)
		ctrl.PostClick(1246, int32(y)).Wait()
		time.Sleep(1000 * time.Millisecond)

		// OCR scan after scrolling.
		result, ok := nestScanPage(ctx)
		if ok {
			log.Info().
				Int("page", page).
				Str("component", "NestScroll").
				Msg("found incomplete nest after scroll")
			return result, true
		}

		// Check if we've reached the bottom (same OCR result as last time).
		currentText := nestOCRText(ctx)
		if currentText != "" && currentText == lastOCRText {
			log.Info().
				Int("page", page).
				Str("component", "NestScroll").
				Msg("reached end of list (same OCR result)")
			return nil, false
		}
		lastOCRText = currentText
	}

	log.Info().
		Int("max_pages", param.MaxPages).
		Str("component", "NestScroll").
		Msg("max pages reached, no incomplete nest found")
	return nil, false
}

// nestScanPage runs OCR on the current page and checks for incomplete nests.
func nestScanPage(ctx *maa.Context) (*maa.CustomRecognitionResult, bool) {
	detail, err := ctx.RunRecognition(
		"__NestScroll_OCR",
		nil,
		`{
			"__NestScroll_OCR": {
				"recognition": "OCR",
				"roi": [460, 95, 780, 560]
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		return nil, false
	}

	denominators := []int{24, 36, 48}
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
			if y < 110 {
				y = 180
			}
			box := maa.Rect{1152, y - 1, 2, 2}
			return &maa.CustomRecognitionResult{
				Box:    box,
				Detail: fmt.Sprintf(`{"current":%d,"total":%d}`, current, total),
			}, true
		}
	}

	return nil, false
}

// nestOCRText returns the concatenated OCR text of the current page for end-of-list detection.
func nestOCRText(ctx *maa.Context) string {
	detail, err := ctx.RunRecognition(
		"__NestScroll_OCREnd",
		nil,
		`{
			"__NestScroll_OCREnd": {
				"recognition": "OCR",
				"roi": [460, 95, 780, 560]
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		return ""
	}
	return detail.DetailJson
}

// ---------------------------------------------------------------------------
// ScrollBookAction — replaced by NestScrollRecognition (CustomRecognition).
// Kept as a compatibility shim that delegates to the recognition loop.
// ---------------------------------------------------------------------------

type ScrollBookAction struct{}

var _ maa.CustomActionRunner = &ScrollBookAction{}

func (a *ScrollBookAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	_, ok := (&NestScrollRecognition{}).Run(ctx, &maa.CustomRecognitionArg{
		CustomRecognitionParam: `{"max_pages":5}`,
	})
	return ok
}

// ---------------------------------------------------------------------------
// ApproachNestAction — walks forward and interacts with F prompts to enter
// the nest combat area.
// ---------------------------------------------------------------------------

type ApproachNestAction struct{}

var _ maa.CustomActionRunner = &ApproachNestAction{}

func (a *ApproachNestAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	ctrl := ctx.GetTasker().GetController()
	ctrl.PostKeyDown(13).Wait()
	defer ctrl.PostKeyUp(13).Wait()

	for i := 0; i < 24; i++ {
		if ctx.GetTasker().Stopping() {
			return true
		}
		if a.inCombat(ctx) {
			return true
		}
		if a.hasFPrompt(ctx) {
			ctrl.PostKeyUp(13).Wait()
			ctrl.PostClickKey(3).Wait()
			time.Sleep(1500 * time.Millisecond)
			ctrl.PostKeyDown(13).Wait()
		}
		time.Sleep(500 * time.Millisecond)
	}
	return true
}

func (a *ApproachNestAction) inCombat(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition(
		"__NightmareApproach_Combat",
		nil,
		`{
			"__NightmareApproach_Combat": {
				"recognition": "TemplateMatch",
				"template": "has_target.png",
				"threshold": 0.6,
				"roi": [400, 200, 800, 600]
			}
		}`,
	)
	return err == nil && detail != nil && detail.Hit
}

func (a *ApproachNestAction) hasFPrompt(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition(
		"__NightmareApproach_F",
		nil,
		`{
			"__NightmareApproach_F": {
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

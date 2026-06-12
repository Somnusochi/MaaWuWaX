// Package pickup implements auto-pickup Custom Recognition and Action for Wuthering Waves.
package pickup

import (
	"fmt"
	"image"
	"strings"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

// PickTextFilterRecognition filters pickup text against blacklist/whitelist.
type PickTextFilterRecognition struct{}

var _ maa.CustomRecognitionRunner = &PickTextFilterRecognition{}

var (
	defaultBlacklist = []string{"开始合成", "领取奖励", "Claim", "合成台"}
	defaultWhitelist = []string{"吸收", "Absorb"}
)

type pickTextFilterParam struct {
	Whitelist string `json:"whitelist"`
	Blacklist string `json:"blacklist"`
}

func parsePickFilterParam(raw string) (wl, bl []string) {
	if raw == "" {
		return defaultWhitelist, defaultBlacklist
	}
	var p pickTextFilterParam
	if err := sonic.Unmarshal([]byte(raw), &p); err == nil {
		return splitPickCSV(p.Whitelist), splitPickCSV(p.Blacklist)
	}
	return defaultWhitelist, defaultBlacklist
}

func splitPickCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func (r *PickTextFilterRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	whitelist, blacklist := parsePickFilterParam(arg.CustomRecognitionParam)

	// 1. Find F-key prompt (primary template, high threshold like ok-ww 0.8)
	fDetail, err := ctx.RunRecognition("AutoPick_FPromptPrimary", arg.Img)
	if err != nil || fDetail == nil || !fDetail.Hit {
		fDetail, err = ctx.RunRecognition("AutoPick_FPromptFallback", arg.Img)
		if err != nil || fDetail == nil || !fDetail.Hit {
			return nil, false
		}
	}

	// 2. Check white color percentage on F icon (ok-ww: white_color_percentage > 0.5)
	whitePct := checkFIconWhitePct(arg.Img, fDetail.Box)
	if whitePct < 0.5 {
		log.Debug().Float64("white_pct", whitePct).Str("component", "PickTextFilter").Msg("F icon white ratio too low, skip")
		return nil, false
	}

	// 3. Detect dialog_3_dots to determine if this is a dialog interaction (ok-ww pattern)
	hasDialog, _ := ctx.RunRecognition("AutoPick_MessageDialog", arg.Img)
	isDialog := hasDialog != nil && hasDialog.Hit

	// 4. OCR the interaction text
	ocrDetail, err := ctx.RunRecognition("AutoPick_TextOCR", arg.Img)
	text := ""
	if err == nil && ocrDetail != nil && ocrDetail.Hit {
		text = ocrDetail.DetailJson
	}

	// 5. Decision logic (ok-ww matching)
	//    - Dialog + whitelist match → allow pickup
	//    - Dialog + no whitelist → reject
	//    - No dialog + no blacklist → allow pickup (generic)
	//    - Blacklist match → reject

	if isDialog {
		// Dialog mode: must match whitelist
		for _, word := range whitelist {
			if word != "" && strings.Contains(text, word) {
				return &maa.CustomRecognitionResult{
					Box:    fDetail.Box,
					Detail: fmt.Sprintf(`{"text":%q,"mode":"dialog_whitelist"}`, text),
				}, true
			}
		}
		log.Debug().Str("text", text).Str("component", "PickTextFilter").Msg("dialog mode, no whitelist match")
		return nil, false
	}

	// Non-dialog mode: check blacklist
	for _, word := range blacklist {
		if word != "" && strings.Contains(text, word) {
			log.Debug().Str("text", text).Str("matched", word).Str("component", "PickTextFilter").Msg("blacklisted")
			return nil, false
		}
	}

	if text == "" {
		// No OCR text but F icon with good white ratio → allow generic pickup
		return &maa.CustomRecognitionResult{
			Box:    fDetail.Box,
			Detail: `{"text":"","mode":"generic"}`,
		}, true
	}

	// Whitelist match in non-dialog mode → still allow
	for _, word := range whitelist {
		if word != "" && strings.Contains(text, word) {
			return &maa.CustomRecognitionResult{
				Box:    fDetail.Box,
				Detail: fmt.Sprintf(`{"text":%q,"mode":"whitelist"}`, text),
			}, true
		}
	}

	// Generic allow (non-dialog, non-blacklisted)
	return &maa.CustomRecognitionResult{
		Box:    fDetail.Box,
		Detail: fmt.Sprintf(`{"text":%q,"mode":"generic"}`, text),
	}, true
}

// checkFIconWhitePct approximates ok-ww's white_color_percentage check.
// Returns the ratio of near-white pixels in a sampled region around the F icon.
func checkFIconWhitePct(img image.Image, box maa.Rect) float64 {
	if img == nil || box[2] <= 0 || box[3] <= 0 {
		return 1.0 // can't check, allow
	}
	bounds := img.Bounds()
	x1 := int(box[0])
	y1 := int(box[1])
	x2 := x1 + int(box[2])
	y2 := y1 + int(box[3])
	if x1 < bounds.Min.X {
		x1 = bounds.Min.X
	}
	if y1 < bounds.Min.Y {
		y1 = bounds.Min.Y
	}
	if x2 > bounds.Max.X {
		x2 = bounds.Max.X
	}
	if y2 > bounds.Max.Y {
		y2 = bounds.Max.Y
	}
	if x2 <= x1 || y2 <= y1 {
		return 1.0
	}
	total := 0
	white := 0
	threshold := uint32(235 << 8) // RGB threshold for "near white"
	for y := y1; y < y2; y += 2 {
		for x := x1; x < x2; x += 2 {
			r, g, b, _ := img.At(x, y).RGBA()
			if (r>>8) > threshold>>8 && (g>>8) > threshold>>8 && (b>>8) > threshold>>8 {
				white++
			}
			total++
		}
	}
	if total == 0 {
		return 1.0
	}
	return float64(white) / float64(total)
}

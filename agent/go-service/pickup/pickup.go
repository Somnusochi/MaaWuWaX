// Package pickup implements auto-pickup Custom Recognition and Action for Wuthering Waves.
package pickup

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
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
	fDetail, err := ctx.RunRecognition(
		"__PickTextFilter_FindF",
		arg.Img,
		`{
			"__PickTextFilter_FindF": {
				"recognition": "TemplateMatch",
				"template": "pick_up_f_hcenter_vcenter.png",
				"threshold": 0.75
			}
		}`,
	)
	if err != nil || fDetail == nil || !fDetail.Hit {
		fDetail, err = ctx.RunRecognition(
			"__PickTextFilter_FindF_Fallback",
			arg.Img,
			`{
				"__PickTextFilter_FindF_Fallback": {
					"recognition": "TemplateMatch",
					"template": "pick_up_f.png",
					"threshold": 0.65,
					"roi": [300, 200, 680, 480]
				}
			}`,
		)
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
	hasDialog, _ := ctx.RunRecognition(
		"__PickTextFilter_3Dots",
		arg.Img,
		`{
			"__PickTextFilter_3Dots": {
				"recognition": "TemplateMatch",
				"template": "message_dialog.png",
				"threshold": 0.45
			}
		}`,
	)
	isDialog := hasDialog != nil && hasDialog.Hit

	// 4. OCR the interaction text
	ocrDetail, err := ctx.RunRecognition(
		"PickTextFilter_OCR",
		arg.Img,
		`{
			"PickTextFilter_OCR": {
				"recognition": "OCR",
				"roi": [300, 200, 680, 100]
			}
		}`,
	)
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

// ---------------------------------------------------------------------------
// PickEnhancedAction — enhanced pickup: press F multiple times with whitelist check.
// ---------------------------------------------------------------------------

type PickEnhancedAction struct{}

var _ maa.CustomActionRunner = &PickEnhancedAction{}

type pickEnhancedParam struct {
	MaxAttempts int `json:"max_attempts"`
}

func (a *PickEnhancedAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := pickEnhancedParam{MaxAttempts: 5}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "PickEnhanced").Msg("failed to parse param")
		}
	}

	ctrl := ctx.GetTasker().GetController()
	fCode := keycode.MustCode("F")
	picked := 0

	for i := 0; i < param.MaxAttempts; i++ {
		if ctx.GetTasker().Stopping() {
			break
		}

		// Check if F icon is still visible.
		detail, err := ctx.RunRecognition(
			fmt.Sprintf("__PickEnhanced_Check_%d", i),
			nil,
			`{
				"__PickEnhanced_Check": {
					"recognition": "TemplateMatch",
					"template": "pick_up_f.png",
					"threshold": 0.65,
					"roi": [300, 200, 680, 480]
				}
			}`,
		)
		if err != nil || detail == nil || !detail.Hit {
			log.Debug().Str("component", "PickEnhanced").Msg("no more F icons")
			break
		}

		ctrl.PostClickKey(fCode).Wait()
		time.Sleep(300 * time.Millisecond)
		picked++
	}

	log.Info().
		Str("component", "PickEnhanced").
		Int("picked", picked).
		Msg("enhanced pickup done")

	return true
}

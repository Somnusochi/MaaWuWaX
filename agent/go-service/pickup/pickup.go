// Package pickup implements auto-pickup Custom Recognition and Action for Wuthering Waves.
package pickup

import (
	"fmt"
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
	blacklist = []string{"开始合成", "领取奖励", "Claim", "合成台"}
	whitelist = []string{"吸收", "Absorb"}
)

func (r *PickTextFilterRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	// Run OCR on the F-key area.
	detail, err := ctx.RunRecognition(
		"PickTextFilter_OCR",
		arg.Img,
		fmt.Sprintf(`{
			"PickTextFilter_OCR": {
				"recognition": "OCR",
				"roi": [300, 200, 680, 100]
			}
		}`),
	)
	if err != nil || detail == nil || !detail.Hit {
		log.Debug().Err(err).Str("component", "PickTextFilter").Msg("OCR failed or no hit")
		return nil, false
	}

	text := detail.DetailJson
	if text == "" {
		return nil, false
	}

	// Check blacklist first.
	for _, word := range blacklist {
		if strings.Contains(text, word) {
			log.Debug().Str("text", text).Str("matched", word).Msg("blacklisted")
			return nil, false
		}
	}

	// Check whitelist.
	for _, word := range whitelist {
		if strings.Contains(text, word) {
			return &maa.CustomRecognitionResult{
				Box:    maa.Rect{300, 200, 680, 100},
				Detail: fmt.Sprintf(`{"text":%q}`, text),
			}, true
		}
	}

	return nil, false
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

// Package stamina implements stamina OCR reading for Wuthering Waves.
package stamina

import (
	"fmt"
	"strconv"
	"strings"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// StaminaReader — OCR reads current stamina value from the F2 book screen.
// Returns the current and total stamina as JSON detail.
// ---------------------------------------------------------------------------

type StaminaReader struct{}

var _ maa.CustomRecognitionRunner = &StaminaReader{}

func (r *StaminaReader) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	// OCR the stamina text area (e.g., "120/240").
	detail, err := ctx.RunRecognition("Stamina_OCR", arg.Img)
	if err != nil || detail == nil || !detail.Hit {
		log.Debug().Str("component", "StaminaReader").Msg("stamina OCR failed or no hit")
		return nil, false
	}

	text := detail.DetailJson
	current, total := parseStaminaText(text)
	if total <= 0 {
		log.Debug().Str("component", "StaminaReader").Str("text", text).Msg("could not parse stamina")
		return nil, false
	}

	log.Info().
		Str("component", "StaminaReader").
		Int("current", current).
		Int("total", total).
		Msg("stamina read")

	return &maa.CustomRecognitionResult{
		Box:    detail.Box,
		Detail: fmt.Sprintf(`{"current":%d,"total":%d}`, current, total),
	}, true
}

// parseStaminaText extracts current and total from text like "120/240".
func parseStaminaText(text string) (current, total int) {
	// Strip JSON quotes if present.
	text = strings.Trim(text, `"`)
	text = strings.TrimSpace(text)

	parts := strings.SplitN(text, "/", 2)
	if len(parts) != 2 {
		return 0, 0
	}

	c, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	t, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	return c, t
}

// ---------------------------------------------------------------------------
// StaminaCheckRecognition — checks if stamina is above a threshold.
// Param: {"min_stamina": 60}
// ---------------------------------------------------------------------------

type StaminaCheckRecognition struct{}

var _ maa.CustomRecognitionRunner = &StaminaCheckRecognition{}

type staminaCheckParam struct {
	MinStamina int `json:"min_stamina"`
}

func (r *StaminaCheckRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := staminaCheckParam{MinStamina: 60}
	if arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "StaminaCheck").Msg("failed to parse param")
		}
	}

	// Read stamina via StaminaReader logic.
	detail, err := ctx.RunRecognition("Stamina_OCR", arg.Img)
	if err != nil || detail == nil || !detail.Hit {
		return nil, false
	}

	current, _ := parseStaminaText(detail.DetailJson)
	if current >= param.MinStamina {
		return &maa.CustomRecognitionResult{
			Box:    detail.Box,
			Detail: fmt.Sprintf(`{"current":%d,"sufficient":true}`, current),
		}, true
	}

	return nil, false
}

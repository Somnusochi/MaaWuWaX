// Package daily implements daily-routine Custom Actions for Wuthering Waves.
package daily

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

// DailyNeedsStaminaRecognition gates stamina routing based on task parameters.
type DailyNeedsStaminaRecognition struct{}

var _ maa.CustomRecognitionRunner = &DailyNeedsStaminaRecognition{}

func (r *DailyNeedsStaminaRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	// The OCR check for progress < 180 and reward points < 100 has already passed in Pipeline JSON.
	// We only need to verify if the task actually wants to spend stamina.
	type staminaParam struct {
		StaminaType string `json:"stamina_type"`
	}
	param := staminaParam{StaminaType: "none"}
	if arg != nil && arg.CustomRecognitionParam != "" {
		_ = sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param)
	}
	if param.StaminaType == "" || param.StaminaType == "none" {
		return nil, false
	}
	return &maa.CustomRecognitionResult{Box: maa.Rect{0, 0, 1, 1}}, true
}

type nightmareModeParam struct {
	NightmareMode string `json:"nightmare_mode"`
	StaminaType   string `json:"stamina_type"`
}

// DailyNeedsNightmareRecognition gates optional NightmareNest routing based on params.
type DailyNeedsNightmareRecognition struct{}

var _ maa.CustomRecognitionRunner = &DailyNeedsNightmareRecognition{}

func (r *DailyNeedsNightmareRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := nightmareModeParam{NightmareMode: "none"}
	if arg != nil && arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "DailyNeedsNightmare").Msg("failed to parse param")
		}
	}
	if param.NightmareMode == "" || param.NightmareMode == "none" {
		return nil, false
	}
	if param.NightmareMode == "capture" && param.StaminaType == "tacet" {
		log.Info().
			Str("component", "DailyNeedsNightmare").
			Str("mode", param.NightmareMode).
			Str("stamina_type", param.StaminaType).
			Msg("skip capture-mode nightmare because tacet route already covers daily echo")
		return nil, false
	}

	return &maa.CustomRecognitionResult{Box: maa.Rect{0, 0, 1, 1}}, true
}

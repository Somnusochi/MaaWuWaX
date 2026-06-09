// Package dialogskip implements dialog-skip Custom Recognition for Wuthering Waves.
// The basic dialog skip logic (skip button, auto-play, arrow click) is handled by Pipeline JSON.
// This package provides additional Go-level detection for edge cases.
package dialogskip

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

// IsMailEnabledRecognition is a stub that always returns enabled=true.
// In production, this should check whether the mail icon has a red dot or is visible.
type IsMailEnabledRecognition struct{}

var _ maa.CustomRecognitionRunner = &IsMailEnabledRecognition{}

func (r *IsMailEnabledRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	// TODO: Actually check mail icon visibility / red dot.
	log.Debug().Str("component", "IsMailEnabled").Msg("returning enabled=true (stub)")
	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"enabled":true}`,
	}, true
}

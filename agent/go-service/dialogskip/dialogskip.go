// Package dialogskip — ok-ww port: SkipDialogTask + SkipBaseTask.
// Pipeline handles the base flow; Go handles detection edge cases.
package dialogskip

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
)

// IsMailEnabledRecognition — ok-ww: checks for red notification dot on mail icon.
type IsMailEnabledRecognition struct{}

var _ maa.CustomRecognitionRunner = &IsMailEnabledRecognition{}

func (r *IsMailEnabledRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	// ok-ww: check for red_dot template on mail icon (threshold 0.6)
	detail, err := ctx.RunRecognition("__Mail_RedDot", arg.Img, `{
		"__Mail_RedDot": {
			"recognition": "TemplateMatch",
			"template": "red_dot.png",
			"threshold": 0.6,
			"roi": [810, 640, 30, 30]
		}
	}`)
	enabled := err == nil && detail != nil && detail.Hit

	if enabled {
		return &maa.CustomRecognitionResult{
			Box:    maa.Rect{810, 640, 30, 30},
			Detail: `{"enabled":true}`,
		}, true
	}
	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"enabled":false}`,
	}, false
}

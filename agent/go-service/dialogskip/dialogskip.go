// Package dialogskip — ok-ww port: SkipDialogTask + SkipBaseTask.
// Pipeline handles the base flow; Go handles detection edge cases.
package dialogskip

import (
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
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

type SkipDialogAdvancedAction struct{}

var _ maa.CustomActionRunner = &SkipDialogAdvancedAction{}

type templateClick struct {
	name      string
	template  string
	threshold float64
}

func (a *SkipDialogAdvancedAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	clicks := []templateClick{
		{name: "__Skip_ClickSkipNew", template: "skip_dialog_new.png", threshold: 0.72},
		{name: "__Skip_ClickSkip", template: "skip_dialog.png", threshold: 0.72},
		{name: "__Skip_ClickBtn", template: "skip_btn.png", threshold: 0.7},
		{name: "__Skip_ConfirmNew", template: "skip_quest_confirm_new.png", threshold: 0.75},
		{name: "__Skip_Confirm", template: "skip_quest_confirm.png", threshold: 0.75},
		{name: "__Skip_DialogConfirm", template: "skip_dialog_confirm.png", threshold: 0.7},
		{name: "__Skip_DialogCheck", template: "skip_dialog_check.png", threshold: 0.7},
		{name: "__Skip_Close", template: "btn_dialog_close.png", threshold: 0.8},
		{name: "__Skip_AutoPlay", template: "btn_auto_play_dialog.png", threshold: 0.7},
		{name: "__Skip_Arrow", template: "btn_dialog_arrow.png", threshold: 0.65},
		{name: "__Skip_Dots", template: "btn_dialog_3dots.png", threshold: 0.65},
		{name: "__Skip_ArrowLegacy", template: "dialog_arrow.png", threshold: 0.6},
		{name: "__Skip_DotsLegacy", template: "dialog_3_dots.png", threshold: 0.6},
	}

	for _, item := range clicks {
		if clickTemplate(ctx, item) {
			return true
		}
	}

	if clickMessageDialog(ctx) {
		return true
	}

	log.Debug().Str("component", "SkipDialog").Msg("no dialog skip target found")
	return true
}

func clickTemplate(ctx *maa.Context, item templateClick) bool {
	detail, err := ctx.RunRecognition(
		item.name,
		nil,
		`{
			"`+item.name+`": {
				"recognition": "TemplateMatch",
				"template": "`+item.template+`",
				"threshold": `+formatFloat(item.threshold)+`
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		return false
	}

	box := detail.Box
	ctx.GetTasker().GetController().PostClick(int32(box[0]+box[2]/2), int32(box[1]+box[3]/2)).Wait()
	time.Sleep(200 * time.Millisecond)
	log.Info().Str("component", "SkipDialog").Str("template", item.template).Msg("clicked dialog target")
	return true
}

func clickMessageDialog(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition(
		"__Skip_MessageDialog",
		nil,
		`{
			"__Skip_MessageDialog": {
				"recognition": "TemplateMatch",
				"template": "message_dialog.png",
				"threshold": 0.65
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		return false
	}

	box := detail.Box
	x := int32(box[0] + box[2]/2)
	y := int32(box[1] + box[3]*3)
	if y > 710 {
		y = 710
	}
	ctx.GetTasker().GetController().PostClick(x, y).Wait()
	time.Sleep(200 * time.Millisecond)
	log.Info().Str("component", "SkipDialog").Msg("clicked message dialog continuation area")
	return true
}

func formatFloat(value float64) string {
	switch value {
	case 0.6:
		return "0.6"
	case 0.65:
		return "0.65"
	case 0.7:
		return "0.7"
	case 0.72:
		return "0.72"
	case 0.75:
		return "0.75"
	case 0.8:
		return "0.8"
	default:
		return "0.7"
	}
}

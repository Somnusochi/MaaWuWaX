// Package dialogskip — ok-ww port: SkipDialogTask + SkipBaseTask.
// Pipeline handles the base flow; Go handles detection edge cases.
package dialogskip

import (
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

type SkipDialogAdvancedAction struct{}
type SkipDialogMessageClickAction struct{}

var _ maa.CustomActionRunner = &SkipDialogAdvancedAction{}
var _ maa.CustomActionRunner = &SkipDialogMessageClickAction{}

type templateClick struct {
	node string
}

func (a *SkipDialogAdvancedAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	clicks := []templateClick{
		{node: "SkipDialog_ClickSkipNew"},
		{node: "SkipDialog_ClickSkip"},
		{node: "SkipDialog_ClickBtn"},
		{node: "SkipDialog_ConfirmNew"},
		{node: "SkipDialog_Confirm"},
		{node: "SkipDialog_DialogConfirm"},
		{node: "SkipDialog_DialogCheck"},
		{node: "SkipDialog_Close"},
		{node: "SkipDialog_AutoPlay"},
		{node: "SkipDialog_Arrow"},
		{node: "SkipDialog_Dots"},
		{node: "SkipDialog_ArrowLegacy"},
		{node: "SkipDialog_DotsLegacy"},
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
	detail, err := ctx.RunRecognition(item.node, nil)
	if err != nil || detail == nil || !detail.Hit {
		return false
	}

	box := detail.Box
	ctx.GetTasker().GetController().PostClick(int32(box[0]+box[2]/2), int32(box[1]+box[3]/2)).Wait()
	time.Sleep(200 * time.Millisecond)
	log.Info().Str("component", "SkipDialog").Str("node", item.node).Msg("clicked dialog target")
	return true
}

func clickMessageDialog(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition("SkipDialog_MessageDialog", nil)
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

func (a *SkipDialogMessageClickAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	return clickMessageDialog(ctx)
}

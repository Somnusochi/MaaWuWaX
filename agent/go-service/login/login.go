// Package login implements auto-login Custom Recognition for Wuthering Waves.
package login

import (
	"fmt"
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

// LoginScreenDetect recognizes login screen elements and clicks them.
type LoginScreenDetect struct{}

var _ maa.CustomRecognitionRunner = &LoginScreenDetect{}

var centerROI = [4]int{384, 216, 512, 288}
var bottomRightROI = [4]int{760, 430, 500, 260}

func (r *LoginScreenDetect) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	ctrl := ctx.GetTasker().GetController()

	if detail, ok := r.clickTemplate(ctx, arg, "monthly_card.png", 0.65, "monthly_card"); ok {
		ctrl.PostClick(640, 641).Wait()
		time.Sleep(1500 * time.Millisecond)
		ctrl.PostClick(640, 641).Wait()
		return detail, true
	}

	if detail, ok := r.clickTemplate(ctx, arg, "login_close.png", 0.7, "login_close"); ok {
		return detail, true
	}

	for _, text := range []string{"登录", "Login", "登入"} {
		if r.hasOCR(ctx, arg, "+86", centerROI) {
			return nil, false
		}
		if detail, ok := r.clickOCR(ctx, arg, text, centerROI, "login"); ok {
			return detail, true
		}
	}

	if r.hasOCR(ctx, arg, "隐私", centerROI) || r.hasOCR(ctx, arg, "Privacy", centerROI) {
		for _, text := range []string{"同意", "Agree"} {
			if detail, ok := r.clickOCR(ctx, arg, text, centerROI, "agree"); ok {
				return detail, true
			}
		}
	}

	for _, text := range []string{"游戏即将重启", "遊戲即將重啟"} {
		if r.hasOCR(ctx, arg, text, [4]int{0, 0, 1280, 720}) {
			for _, confirm := range []string{"确认", "確認", "Confirm"} {
				if detail, ok := r.clickOCR(ctx, arg, confirm, [4]int{0, 0, 1280, 720}, "restart_confirm"); ok {
					return detail, true
				}
			}
		}
	}

	for _, text := range []string{"开始游戏", "进入游戏", "Tap to Start"} {
		if r.hasOCR(ctx, arg, "登录", bottomRightROI) || r.hasOCR(ctx, arg, "Login", bottomRightROI) {
			return nil, false
		}
		if detail, ok := r.clickOCR(ctx, arg, text, bottomRightROI, "start"); ok {
			return detail, true
		}
	}

	if _, ok := r.findTemplate(ctx, arg, "switch_account.png", 0.7); ok {
		for _, text := range []string{"登录", "Login", "登入"} {
			if r.hasOCR(ctx, arg, text, [4]int{474, 454, 332, 266}) {
				ctrl.PostClick(644, 667).Wait()
				time.Sleep(3 * time.Second)
				return &maa.CustomRecognitionResult{
					Box:    maa.Rect{474, 454, 332, 266},
					Detail: `{"action":"switch_account_login"}`,
				}, true
			}
		}
	}

	return nil, false
}

func (r *LoginScreenDetect) clickTemplate(ctx *maa.Context, arg *maa.CustomRecognitionArg, template string, threshold float64, action string) (*maa.CustomRecognitionResult, bool) {
	detail, ok := r.findTemplate(ctx, arg, template, threshold)
	if !ok {
		return nil, false
	}

	box := detail.Box
	ctrl := ctx.GetTasker().GetController()
	ctrl.PostClick(int32(box[0]+box[2]/2), int32(box[1]+box[3]/2)).Wait()
	time.Sleep(800 * time.Millisecond)
	log.Info().Str("component", "LoginScreen").Str("action", action).Str("template", template).Msg("clicked template")
	return &maa.CustomRecognitionResult{Box: box, Detail: fmt.Sprintf(`{"action":%q}`, action)}, true
}

func (r *LoginScreenDetect) findTemplate(ctx *maa.Context, arg *maa.CustomRecognitionArg, template string, threshold float64) (*maa.RecognitionDetail, bool) {
	detail, err := ctx.RunRecognition(
		"LoginScreenDetect_Template",
		arg.Img,
		fmt.Sprintf(`{
			"LoginScreenDetect_Template": {
				"recognition": "TemplateMatch",
				"template": %q,
				"threshold": %.2f
			}
		}`, template, threshold),
	)
	return detail, err == nil && detail != nil && detail.Hit
}

func (r *LoginScreenDetect) clickOCR(ctx *maa.Context, arg *maa.CustomRecognitionArg, text string, roi [4]int, action string) (*maa.CustomRecognitionResult, bool) {
	detail, ok := r.findOCR(ctx, arg, text, roi)
	if !ok {
		return nil, false
	}

	box := detail.Box
	ctrl := ctx.GetTasker().GetController()
	ctrl.PostClick(int32(box[0]+box[2]/2), int32(box[1]+box[3]/2)).Wait()
	time.Sleep(1000 * time.Millisecond)
	log.Info().Str("component", "LoginScreen").Str("action", action).Str("text", text).Msg("clicked OCR")
	return &maa.CustomRecognitionResult{Box: box, Detail: fmt.Sprintf(`{"action":%q,"text":%q}`, action, text)}, true
}

func (r *LoginScreenDetect) hasOCR(ctx *maa.Context, arg *maa.CustomRecognitionArg, text string, roi [4]int) bool {
	_, ok := r.findOCR(ctx, arg, text, roi)
	return ok
}

func (r *LoginScreenDetect) findOCR(ctx *maa.Context, arg *maa.CustomRecognitionArg, text string, roi [4]int) (*maa.RecognitionDetail, bool) {
	detail, err := ctx.RunRecognition(
		"LoginScreenDetect_OCR",
		arg.Img,
		fmt.Sprintf(`{
			"LoginScreenDetect_OCR": {
				"recognition": "OCR",
				"expected": %q,
				"roi": [%d, %d, %d, %d]
			}
		}`, text, roi[0], roi[1], roi[2], roi[3]),
	)
	return detail, err == nil && detail != nil && detail.Hit
}

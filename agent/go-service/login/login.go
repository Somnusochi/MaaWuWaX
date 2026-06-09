// Package login implements auto-login Custom Recognition for Wuthering Waves.
package login

import (
	"fmt"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

// LoginScreenDetect recognizes login screen elements and clicks them.
type LoginScreenDetect struct{}

var _ maa.CustomRecognitionRunner = &LoginScreenDetect{}

// candidates are the OCR texts to look for on login screens.
var candidates = []string{
	"开始游戏", "进入游戏", "点击开始",
	"Tap to Start",
	"确认", "确定", "Confirm", "OK",
	"同意", "Agree",
	"登录", "Login", "进入",
	"关闭", "Close",
	"取消", "Cancel",
}

func (r *LoginScreenDetect) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	ctrl := ctx.GetTasker().GetController()

	for _, text := range candidates {
		detail, err := ctx.RunRecognition(
			"LoginScreenDetect_OCR",
			arg.Img,
			fmt.Sprintf(`{
				"LoginScreenDetect_OCR": {
					"recognition": "OCR",
					"expected": %q
				}
			}`, text),
		)
		if err != nil {
			continue
		}
		if detail != nil && detail.Hit {
			// Click the detected element.
			box := detail.Box
			cx := int32(box[0] + box[2]/2)
			cy := int32(box[1] + box[3]/2)
			ctrl.PostClick(cx, cy).Wait()

			log.Info().
				Str("component", "LoginScreen").
				Str("text", text).
				Msg("detected and clicked")

			return &maa.CustomRecognitionResult{
				Box:    box,
				Detail: fmt.Sprintf(`{"text":%q}`, text),
			}, true
		}
	}

	return nil, false
}

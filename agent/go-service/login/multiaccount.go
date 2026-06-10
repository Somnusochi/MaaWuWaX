package login

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

type MultiAccountSwitchAction struct {
	switchCount int
	done        map[string]bool
	failed      int
}

var _ maa.CustomActionRunner = &MultiAccountSwitchAction{}

type multiAccountParam struct {
	MaxAccounts int `json:"max_accounts"`
	MaxFailures int `json:"max_failures"`
}

type accountChoice struct {
	Name string
	Box  maa.Rect
}

var maskedAccountRe = regexp.MustCompile(`\*{4,}`)

type MultiAccountMarkFailedAction struct{}

var _ maa.CustomActionRunner = &MultiAccountMarkFailedAction{}

func (a *MultiAccountSwitchAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := multiAccountParam{MaxAccounts: 2}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "MultiAccountSwitch").Msg("failed to parse param")
		}
	}
	if param.MaxAccounts < 1 {
		param.MaxAccounts = 1
	}
	if a.done == nil {
		a.done = map[string]bool{}
	}
	if param.MaxFailures <= 0 {
		param.MaxFailures = param.MaxAccounts
	}
	if a.failed >= param.MaxFailures {
		log.Warn().
			Str("component", "MultiAccountSwitch").
			Int("failed", a.failed).
			Int("max_failures", param.MaxFailures).
			Msg("failure limit reached")
		return false
	}
	if a.switchCount >= param.MaxAccounts-1 {
		log.Info().Str("component", "MultiAccountSwitch").Int("max_accounts", param.MaxAccounts).Msg("all accounts attempted")
		return false
	}

	ctrl := ctx.GetTasker().GetController()
	ctrl.PostClickKey(53).Wait()
	time.Sleep(1500 * time.Millisecond)
	if !a.hasTemplate(ctx, "esc_setting.png", 0.6) {
		ctrl.PostClickKey(53).Wait()
		time.Sleep(1000 * time.Millisecond)
	}

	ctrl.PostClick(51, 691).Wait()
	time.Sleep(1000 * time.Millisecond)
	a.clickConfirm(ctx)
	time.Sleep(3000 * time.Millisecond)

	if current := a.detectCurrentAccount(ctx); current != "" {
		a.done[current] = true
		log.Info().Str("component", "MultiAccountSwitch").Str("account", current).Msg("marked current account done")
	}

	if !a.openAccountDropdown(ctx) {
		log.Warn().Str("component", "MultiAccountSwitch").Msg("failed to open account dropdown")
		return false
	}
	choice, ok := a.selectNextAccount(ctx)
	if !ok {
		log.Info().Str("component", "MultiAccountSwitch").Msg("no unfinished account found")
		return false
	}
	ctrl.PostClick(int32(choice.Box[0]+choice.Box[2]/2), int32(choice.Box[1]+choice.Box[3]/2)).Wait()
	time.Sleep(2000 * time.Millisecond)

	// ok-ww parity: verify selected account matches displayed account with retry.
	confirmed := false
	for retry := 0; retry < 3; retry++ {
		displayed := a.detectCurrentAccount(ctx)
		if displayed == choice.Name {
			log.Info().Str("component", "MultiAccountSwitch").Str("account", choice.Name).Msg("account selection confirmed")
			confirmed = true
			break
		}
		log.Warn().Str("component", "MultiAccountSwitch").
			Str("expected", choice.Name).Str("displayed", displayed).
			Int("retry", retry+1).Msg("account mismatch, retrying")
		time.Sleep(1000 * time.Millisecond)
		// Re-expand dropdown and re-click
		a.openAccountDropdown(ctx)
		choices := a.detectAccounts(ctx)
		for _, c := range choices {
			if c.Name == choice.Name {
				ctrl.PostClick(int32(c.Box[0]+c.Box[2]/2), int32(c.Box[1]+c.Box[3]/2)).Wait()
				time.Sleep(2000 * time.Millisecond)
				break
			}
		}
	}
	if !confirmed {
		log.Warn().Str("component", "MultiAccountSwitch").Str("account", choice.Name).Msg("account confirmation failed, proceeding anyway")
	}

	a.done[choice.Name] = true
	a.switchCount++

	// Click login button after account selection (ok-ww parity).
	a.clickLoginButton(ctx)

	log.Info().
		Str("component", "MultiAccountSwitch").
		Str("account", choice.Name).
		Int("switch_count", a.switchCount).
		Int("max_accounts", param.MaxAccounts).
		Msg("selected next account")
	return true
}

func (a *MultiAccountMarkFailedAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := multiAccountParam{MaxAccounts: 2}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "MultiAccountMarkFailed").Msg("failed to parse param")
		}
	}
	if param.MaxFailures <= 0 {
		param.MaxFailures = param.MaxAccounts
	}

	switcher := multiAccountSwitcher
	if switcher == nil {
		log.Warn().Str("component", "MultiAccountMarkFailed").Msg("switch action state unavailable")
		return true
	}
	switcher.failed++
	log.Warn().
		Str("component", "MultiAccountMarkFailed").
		Int("failed", switcher.failed).
		Int("max_failures", param.MaxFailures).
		Msg("account daily failed, skipping to next account")
	return switcher.failed < param.MaxFailures
}

// clickLoginButton finds and clicks the login/start button after account selection.
func (a *MultiAccountSwitchAction) clickLoginButton(ctx *maa.Context) {
	for _, text := range []string{"登录", "Login", "登入"} {
		detail, err := ctx.RunRecognition(
			"__MultiAccount_LoginBtn",
			nil,
			fmt.Sprintf(`{
				"__MultiAccount_LoginBtn": {
					"recognition": "OCR",
					"expected": %q,
					"roi": [384, 300, 512, 288]
				}
			}`, text),
		)
		if err == nil && detail != nil && detail.Hit {
			box := detail.Box
			ctx.GetTasker().GetController().PostClick(
				int32(box[0]+box[2]/2),
				int32(box[1]+box[3]/2),
			).Wait()
			time.Sleep(3000 * time.Millisecond)
			log.Info().Str("component", "MultiAccountSwitch").Str("text", text).Msg("login button clicked")
			return
		}
	}
	// Fallback: click center-bottom area where login button usually is.
	ctx.GetTasker().GetController().PostClick(640, 409).Wait()
	time.Sleep(3000 * time.Millisecond)
	log.Info().Str("component", "MultiAccountSwitch").Msg("login button fallback clicked")
}

func (a *MultiAccountSwitchAction) openAccountDropdown(ctx *maa.Context) bool {
	for attempt := 0; attempt < 5; attempt++ {
		if a.clickTemplate(ctx, "account_drop_down.png", 0.6) {
			time.Sleep(1000 * time.Millisecond)
		} else if current := a.detectCurrentAccount(ctx); current != "" {
			boxes := a.detectAccounts(ctx)
			for _, choice := range boxes {
				if choice.Name == current {
					ctx.GetTasker().GetController().PostClick(
						int32(choice.Box[0]+choice.Box[2]/2),
						int32(choice.Box[1]+choice.Box[3]/2),
					).Wait()
					time.Sleep(1000 * time.Millisecond)
					break
				}
			}
		}
		if len(a.detectAccounts(ctx)) > 1 {
			return true
		}
	}
	return false
}

func (a *MultiAccountSwitchAction) selectNextAccount(ctx *maa.Context) (accountChoice, bool) {
	choices := a.detectAccounts(ctx)
	for _, choice := range choices {
		if !a.done[choice.Name] {
			return choice, true
		}
	}
	return accountChoice{}, false
}

func (a *MultiAccountSwitchAction) detectCurrentAccount(ctx *maa.Context) string {
	choices := a.detectAccounts(ctx)
	if len(choices) == 1 {
		return choices[0].Name
	}
	return ""
}

func (a *MultiAccountSwitchAction) detectAccounts(ctx *maa.Context) []accountChoice {
	detail, err := ctx.RunRecognition(
		"__MultiAccount_AccountOCR",
		nil,
		`{
			"__MultiAccount_AccountOCR": {
				"recognition": "OCR",
				"roi": [260, 260, 760, 360]
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit || detail.Results == nil {
		return nil
	}
	results := detail.Results.Filtered
	if len(results) == 0 {
		results = detail.Results.All
	}
	choices := make([]accountChoice, 0, len(results))
	for _, result := range results {
		ocr, ok := result.AsOCR()
		if !ok || ocr == nil {
			continue
		}
		match := maskedAccountRe.FindString(ocr.Text)
		if match == "" {
			continue
		}
		name := strings.TrimSpace(ocr.Text)
		choices = append(choices, accountChoice{Name: name, Box: ocr.Box})
	}
	return choices
}

func (a *MultiAccountSwitchAction) hasTemplate(ctx *maa.Context, template string, threshold float64) bool {
	detail, err := ctx.RunRecognition(
		"__MultiAccount_Template",
		nil,
		formatTemplateRecognition("__MultiAccount_Template", template, threshold),
	)
	return err == nil && detail != nil && detail.Hit
}

func (a *MultiAccountSwitchAction) clickTemplate(ctx *maa.Context, template string, threshold float64) bool {
	detail, err := ctx.RunRecognition(
		"__MultiAccount_ClickTemplate",
		nil,
		formatTemplateRecognition("__MultiAccount_ClickTemplate", template, threshold),
	)
	if err != nil || detail == nil || !detail.Hit {
		return false
	}
	ctx.GetTasker().GetController().PostClick(
		int32(detail.Box[0]+detail.Box[2]/2),
		int32(detail.Box[1]+detail.Box[3]/2),
	).Wait()
	return true
}

func (a *MultiAccountSwitchAction) clickConfirm(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition(
		"__MultiAccount_Confirm",
		nil,
		`{
			"__MultiAccount_Confirm": {
				"recognition": "Or",
				"any_of": [
					{"recognition": "TemplateMatch", "template": "btn_yellow_confirm.png", "threshold": 0.7},
					{"recognition": "TemplateMatch", "template": "confirm_btn_hcenter_vcenter.png", "threshold": 0.7},
					{"recognition": "OCR", "expected": "确认"},
					{"recognition": "OCR", "expected": "Confirm"}
				]
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		return false
	}
	ctx.GetTasker().GetController().PostClick(
		int32(detail.Box[0]+detail.Box[2]/2),
		int32(detail.Box[1]+detail.Box[3]/2),
	).Wait()
	return true
}

func formatTemplateRecognition(name string, template string, threshold float64) string {
	return fmt.Sprintf(`{
		%q: {
			"recognition": "TemplateMatch",
			"template": %q,
			"threshold": %.2f
		}
	}`, name, template, threshold)
}

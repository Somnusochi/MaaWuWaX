package login

import (
	"regexp"
	"strings"
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

type MultiAccountState struct {
	switchCount int
	done        map[string]bool
	failed      int
	pending     string
}

type multiAccountParam struct {
	MaxAccounts int `json:"max_accounts"`
	MaxFailures int `json:"max_failures"`
}

type accountChoice struct {
	Name string
	Box  maa.Rect
}

var maskedAccountRe = regexp.MustCompile(`\*{4,}`)

type MultiAccountCanSwitchRecognition struct{}
type MultiAccountMarkCurrentAction struct{}
type MultiAccountSelectNextAction struct{}
type MultiAccountSelectedMatchesRecognition struct{}
type MultiAccountMarkFailedAction struct{}

var _ maa.CustomRecognitionRunner = &MultiAccountCanSwitchRecognition{}
var _ maa.CustomActionRunner = &MultiAccountMarkCurrentAction{}
var _ maa.CustomActionRunner = &MultiAccountSelectNextAction{}
var _ maa.CustomRecognitionRunner = &MultiAccountSelectedMatchesRecognition{}
var _ maa.CustomActionRunner = &MultiAccountMarkFailedAction{}

func (r *MultiAccountCanSwitchRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := multiAccountParam{MaxAccounts: 2}
	if arg != nil && arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "MultiAccountCanSwitch").Msg("failed to parse param")
		}
	}
	if param.MaxAccounts < 1 {
		param.MaxAccounts = 1
	}
	if param.MaxFailures <= 0 {
		param.MaxFailures = param.MaxAccounts
	}

	switcher := multiAccountSwitcher
	if switcher.done == nil {
		switcher.done = map[string]bool{}
	}
	if switcher.failed >= param.MaxFailures {
		log.Warn().
			Str("component", "MultiAccountCanSwitch").
			Int("failed", switcher.failed).
			Int("max_failures", param.MaxFailures).
			Msg("failure limit reached")
		return nil, false
	}
	if switcher.switchCount >= param.MaxAccounts-1 {
		log.Info().Str("component", "MultiAccountCanSwitch").Int("max_accounts", param.MaxAccounts).Msg("all accounts attempted")
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"can_switch":true}`,
	}, true
}

func (a *MultiAccountMarkCurrentAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	switcher := multiAccountSwitcher
	if switcher.done == nil {
		switcher.done = map[string]bool{}
	}
	if current := switcher.detectCurrentAccount(ctx); current != "" {
		switcher.done[current] = true
		log.Info().Str("component", "MultiAccountMarkCurrent").Str("account", current).Msg("marked current account done")
	}
	return true
}

func (a *MultiAccountSelectNextAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	switcher := multiAccountSwitcher
	if switcher.done == nil {
		switcher.done = map[string]bool{}
	}

	choice, ok := switcher.selectNextAccount(ctx)
	if !ok {
		log.Info().Str("component", "MultiAccountSelectNext").Msg("no unfinished account found")
		return false
	}
	ctx.GetTasker().GetController().PostClick(int32(choice.Box[0]+choice.Box[2]/2), int32(choice.Box[1]+choice.Box[3]/2)).Wait()
	time.Sleep(2000 * time.Millisecond)
	switcher.pending = choice.Name

	log.Info().
		Str("component", "MultiAccountSelectNext").
		Str("account", choice.Name).
		Msg("selected candidate account")
	return true
}

func (r *MultiAccountSelectedMatchesRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	switcher := multiAccountSwitcher
	if switcher.pending == "" {
		log.Warn().Str("component", "MultiAccountSelectedMatches").Msg("no pending account")
		return nil, false
	}

	displayed := switcher.detectCurrentAccount(ctx)
	if displayed != switcher.pending {
		log.Warn().
			Str("component", "MultiAccountSelectedMatches").
			Str("expected", switcher.pending).
			Str("displayed", displayed).
			Msg("account mismatch")
		return nil, false
	}

	if switcher.done == nil {
		switcher.done = map[string]bool{}
	}
	switcher.done[switcher.pending] = true
	switcher.switchCount++
	log.Info().
		Str("component", "MultiAccountSelectedMatches").
		Str("account", switcher.pending).
		Int("switch_count", switcher.switchCount).
		Msg("account selection confirmed")
	switcher.pending = ""

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"selected_matches":true}`,
	}, true
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

func (a *MultiAccountState) selectNextAccount(ctx *maa.Context) (accountChoice, bool) {
	if a.pending != "" {
		for _, choice := range a.detectAccounts(ctx) {
			if choice.Name == a.pending {
				return choice, true
			}
		}
	}
	choices := a.detectAccounts(ctx)
	for _, choice := range choices {
		if !a.done[choice.Name] {
			return choice, true
		}
	}
	return accountChoice{}, false
}

func (a *MultiAccountState) detectCurrentAccount(ctx *maa.Context) string {
	choices := a.detectAccounts(ctx)
	if len(choices) == 1 {
		return choices[0].Name
	}
	return ""
}

func (a *MultiAccountState) detectAccounts(ctx *maa.Context) []accountChoice {
	detail, err := ctx.RunRecognition("MultiAccount_AccountOCR", nil)
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

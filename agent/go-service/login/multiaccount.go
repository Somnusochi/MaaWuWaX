package login

import (
	"fmt"
	"regexp"
	"strings"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

const defaultMultiAccountMaxAccounts = 2

type MultiAccountState struct {
	done       map[string]bool
	failed     map[string]bool
	discovered map[string]bool
	pending    string
}

type multiAccountParam struct {
	MaxAccounts int `json:"max_accounts"`
}

type multiAccountAccountListParam struct {
	MinAccounts int `json:"min_accounts"`
}

type accountChoice struct {
	Name string
	Box  maa.Rect
}

var maskedAccountRe = regexp.MustCompile(`\*{4,}`)

type MultiAccountCanSwitchRecognition struct{}
type MultiAccountDropdownExpandedRecognition struct{}
type MultiAccountCurrentVisibleRecognition struct{}
type MultiAccountResetStateAction struct{}
type MultiAccountMarkCurrentAction struct{}
type MultiAccountSelectNextAction struct{}
type MultiAccountSelectedMatchesRecognition struct{}
type MultiAccountMarkFailedAction struct{}

var _ maa.CustomRecognitionRunner = &MultiAccountCanSwitchRecognition{}
var _ maa.CustomRecognitionRunner = &MultiAccountDropdownExpandedRecognition{}
var _ maa.CustomRecognitionRunner = &MultiAccountCurrentVisibleRecognition{}
var _ maa.CustomActionRunner = &MultiAccountResetStateAction{}
var _ maa.CustomActionRunner = &MultiAccountMarkCurrentAction{}
var _ maa.CustomActionRunner = &MultiAccountSelectNextAction{}
var _ maa.CustomRecognitionRunner = &MultiAccountSelectedMatchesRecognition{}
var _ maa.CustomActionRunner = &MultiAccountMarkFailedAction{}

func (r *MultiAccountCanSwitchRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := parseMultiAccountParam(customRecognitionParam(arg), "MultiAccountCanSwitch")

	switcher := multiAccountSwitcher
	switcher.ensureState()
	attempted := switcher.attemptedCount()
	if param.MaxAccounts > 0 && attempted >= param.MaxAccounts {
		log.Info().
			Str("component", "MultiAccountCanSwitch").
			Int("attempted", attempted).
			Int("max_accounts", param.MaxAccounts).
			Msg("reached configured account limit")
		return nil, false
	}
	if remaining := switcher.remainingAccounts(); remaining == 0 && len(switcher.discovered) > 0 {
		log.Info().
			Str("component", "MultiAccountCanSwitch").
			Int("discovered", len(switcher.discovered)).
			Int("done", len(switcher.done)).
			Int("failed", len(switcher.failed)).
			Msg("no unfinished discovered accounts remain")
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"can_switch":true}`,
	}, true
}

func (r *MultiAccountDropdownExpandedRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := multiAccountAccountListParam{MinAccounts: 2}
	if raw := customRecognitionParam(arg); raw != "" {
		if err := sonic.Unmarshal([]byte(raw), &param); err != nil {
			log.Warn().Err(err).Str("component", "MultiAccountDropdownExpanded").Msg("failed to parse param")
		}
	}
	if param.MinAccounts < 2 {
		param.MinAccounts = 2
	}

	choices := multiAccountSwitcher.detectAccounts(ctx)
	if len(choices) < param.MinAccounts {
		log.Warn().
			Str("component", "MultiAccountDropdownExpanded").
			Int("accounts", len(choices)).
			Int("min_accounts", param.MinAccounts).
			Msg("account dropdown not expanded")
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"expanded":true}`,
	}, true
}

func (r *MultiAccountCurrentVisibleRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	current := multiAccountSwitcher.detectCurrentAccount(ctx)
	if current == "" {
		log.Warn().Str("component", "MultiAccountCurrentVisible").Msg("current account not visible on login screen")
		return nil, false
	}

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: fmt.Sprintf(`{"account":%q}`, current),
	}, true
}

func (a *MultiAccountResetStateAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	switcher := multiAccountSwitcher
	switcher.done = map[string]bool{}
	switcher.failed = map[string]bool{}
	switcher.discovered = map[string]bool{}
	switcher.pending = ""
	log.Info().Str("component", "MultiAccountResetState").Msg("reset multi-account state")
	return true
}

func (a *MultiAccountMarkCurrentAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	switcher := multiAccountSwitcher
	switcher.ensureState()
	if current := switcher.detectCurrentAccount(ctx); current != "" {
		switcher.done[current] = true
		delete(switcher.failed, current)
		switcher.pending = ""
		log.Info().Str("component", "MultiAccountMarkCurrent").Str("account", current).Msg("marked current account done")
	}
	return true
}

func (a *MultiAccountSelectNextAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	switcher := multiAccountSwitcher
	switcher.ensureState()

	choice, ok := switcher.selectNextAccount(ctx)
	if !ok {
		log.Info().Str("component", "MultiAccountSelectNext").Msg("no unfinished account found")
		return false
	}
	ctx.GetTasker().GetController().PostClick(int32(choice.Box[0]+choice.Box[2]/2), int32(choice.Box[1]+choice.Box[3]/2)).Wait()
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

	log.Info().
		Str("component", "MultiAccountSelectedMatches").
		Str("account", switcher.pending).
		Msg("account selection confirmed")

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"selected_matches":true}`,
	}, true
}

func (a *MultiAccountMarkFailedAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	switcher := multiAccountSwitcher
	switcher.ensureState()
	if switcher.pending != "" {
		switcher.failed[switcher.pending] = true
		log.Warn().
			Str("component", "MultiAccountMarkFailed").
			Str("account", switcher.pending).
			Int("failed", len(switcher.failed)).
			Msg("account daily failed, marked as failed and skipping")
	}
	switcher.pending = ""
	return true
}

func (a *MultiAccountState) selectNextAccount(ctx *maa.Context) (accountChoice, bool) {
	a.ensureState()
	if a.pending != "" && !a.done[a.pending] && !a.failed[a.pending] {
		for _, choice := range a.detectAccounts(ctx) {
			if choice.Name == a.pending {
				return choice, true
			}
		}
	}
	choices := a.detectAccounts(ctx)
	for _, choice := range choices {
		if !a.done[choice.Name] && !a.failed[choice.Name] {
			return choice, true
		}
	}
	return accountChoice{}, false
}

func (a *MultiAccountState) detectCurrentAccount(ctx *maa.Context) string {
	choices := a.detectAccounts(ctx)
	if len(choices) == 0 {
		return ""
	}
	best := choices[0]
	bestY := best.Box[1] + best.Box[3]/2
	for _, choice := range choices[1:] {
		centerY := choice.Box[1] + choice.Box[3]/2
		if centerY < bestY {
			best = choice
			bestY = centerY
		}
	}
	if len(choices) > 1 {
		log.Debug().
			Str("component", "MultiAccountCurrentVisible").
			Str("selected", best.Name).
			Int("candidates", len(choices)).
			Msg("multiple masked account candidates found, using topmost candidate")
	}
	return best.Name
}

func (a *MultiAccountState) detectAccounts(ctx *maa.Context) []accountChoice {
	a.ensureState()
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
		a.discovered[name] = true
		replaced := false
		for i, existing := range choices {
			if existing.Name != name {
				continue
			}
			existingY := existing.Box[1] + existing.Box[3]/2
			currentY := ocr.Box[1] + ocr.Box[3]/2
			if currentY < existingY {
				choices[i] = accountChoice{Name: name, Box: ocr.Box}
			}
			replaced = true
			break
		}
		if replaced {
			continue
		}
		choices = append(choices, accountChoice{Name: name, Box: ocr.Box})
	}
	return choices
}

func (a *MultiAccountState) ensureState() {
	if a.done == nil {
		a.done = map[string]bool{}
	}
	if a.failed == nil {
		a.failed = map[string]bool{}
	}
	if a.discovered == nil {
		a.discovered = map[string]bool{}
	}
}

func (a *MultiAccountState) attemptedCount() int {
	a.ensureState()
	return len(a.done) + len(a.failed)
}

func (a *MultiAccountState) remainingAccounts() int {
	a.ensureState()
	remaining := 0
	for account := range a.discovered {
		if a.done[account] || a.failed[account] {
			continue
		}
		remaining++
	}
	return remaining
}

func parseMultiAccountParam(raw string, component string) multiAccountParam {
	param := multiAccountParam{MaxAccounts: defaultMultiAccountMaxAccounts}
	if raw != "" {
		if err := sonic.Unmarshal([]byte(raw), &param); err != nil {
			log.Warn().Err(err).Str("component", component).Msg("failed to parse param")
		}
	}
	if param.MaxAccounts < 0 {
		param.MaxAccounts = defaultMultiAccountMaxAccounts
	}
	return param
}

func customRecognitionParam(arg *maa.CustomRecognitionArg) string {
	if arg == nil {
		return ""
	}
	return arg.CustomRecognitionParam
}

func customActionParam(arg *maa.CustomActionArg) string {
	if arg == nil {
		return ""
	}
	return arg.CustomActionParam
}

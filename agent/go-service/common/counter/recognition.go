package counter

import (
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

type counterCheckBelowLimitParam struct {
	CounterKey string `json:"counter_key"`
	MaxCount   int    `json:"max_count"`
}

type counterCheckEqualsParam struct {
	CounterKey string `json:"counter_key"`
	Expected   int    `json:"expected"`
}

type CounterCheckBelowLimitRecognition struct{}
type CounterCheckEqualsRecognition struct{}

var _ maa.CustomRecognitionRunner = &CounterCheckBelowLimitRecognition{}
var _ maa.CustomRecognitionRunner = &CounterCheckEqualsRecognition{}

func (r *CounterCheckBelowLimitRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := counterCheckBelowLimitParam{CounterKey: "default", MaxCount: 1}
	if arg != nil && arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "CounterCheckBelowLimit").Msg("failed to parse param")
		}
	}
	if param.CounterKey == "" {
		param.CounterKey = "default"
	}
	if param.MaxCount <= 0 {
		param.MaxCount = 1
	}

	current := Peek(param.CounterKey)
	if current >= param.MaxCount {
		log.Info().
			Str("component", "CounterCheckBelowLimit").
			Str("counter_key", param.CounterKey).
			Int("current", current).
			Int("max_count", param.MaxCount).
			Msg("counter limit reached")
		return nil, false
	}

	log.Debug().
		Str("component", "CounterCheckBelowLimit").
		Str("counter_key", param.CounterKey).
		Int("current", current).
		Int("max_count", param.MaxCount).
		Msg("counter below limit")
	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"counter_below_limit":true}`,
	}, true
}

func (r *CounterCheckEqualsRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	param := counterCheckEqualsParam{CounterKey: "default", Expected: 0}
	if arg != nil && arg.CustomRecognitionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomRecognitionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "CounterCheckEquals").Msg("failed to parse param")
		}
	}
	if param.CounterKey == "" {
		param.CounterKey = "default"
	}

	current := Peek(param.CounterKey)
	if current != param.Expected {
		log.Debug().
			Str("component", "CounterCheckEquals").
			Str("counter_key", param.CounterKey).
			Int("current", current).
			Int("expected", param.Expected).
			Msg("counter value mismatch")
		return nil, false
	}

	log.Debug().
		Str("component", "CounterCheckEquals").
		Str("counter_key", param.CounterKey).
		Int("current", current).
		Int("expected", param.Expected).
		Msg("counter equals expected")
	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"counter_equals":true}`,
	}, true
}

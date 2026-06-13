package counter

import (
	"sync"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

var counters struct {
	sync.Mutex
	values map[string]int
}

func resetCounter(key string, value int) {
	counters.Lock()
	defer counters.Unlock()
	if counters.values == nil {
		counters.values = map[string]int{}
	}
	counters.values[key] = value
}

func peekCounter(key string) int {
	counters.Lock()
	defer counters.Unlock()
	if counters.values == nil {
		return 0
	}
	return counters.values[key]
}

func incrementCounter(key string, step int) int {
	counters.Lock()
	defer counters.Unlock()
	if counters.values == nil {
		counters.values = map[string]int{}
	}
	counters.values[key] += step
	return counters.values[key]
}

func Reset(key string, value int) {
	if key == "" {
		key = "default"
	}
	resetCounter(key, value)
}

func Peek(key string) int {
	if key == "" {
		key = "default"
	}
	return peekCounter(key)
}

func Increment(key string, step int) int {
	if key == "" {
		key = "default"
	}
	if step <= 0 {
		step = 1
	}
	return incrementCounter(key, step)
}

type counterResetParam struct {
	CounterKey string `json:"counter_key"`
	Value      int    `json:"value"`
}

type counterIncrementParam struct {
	CounterKey string `json:"counter_key"`
	Step       int    `json:"step"`
}

type CounterResetAction struct{}
type CounterIncrementAction struct{}

var _ maa.CustomActionRunner = &CounterResetAction{}
var _ maa.CustomActionRunner = &CounterIncrementAction{}

func (a *CounterResetAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := counterResetParam{CounterKey: "default", Value: 0}
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "CounterReset").Msg("failed to parse param")
		}
	}
	if param.CounterKey == "" {
		param.CounterKey = "default"
	}

	resetCounter(param.CounterKey, param.Value)

	log.Info().
		Str("component", "CounterReset").
		Str("counter_key", param.CounterKey).
		Int("value", param.Value).
		Msg("counter reset")
	return true
}

func (a *CounterIncrementAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := counterIncrementParam{CounterKey: "default", Step: 1}
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "CounterIncrement").Msg("failed to parse param")
		}
	}
	if param.CounterKey == "" {
		param.CounterKey = "default"
	}
	if param.Step <= 0 {
		param.Step = 1
	}

	next := Increment(param.CounterKey, param.Step)
	log.Info().
		Str("component", "CounterIncrement").
		Str("counter_key", param.CounterKey).
		Int("next", next).
		Int("step", param.Step).
		Msg("counter incremented")
	return true
}

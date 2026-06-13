package delay

import (
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

type delayParam struct {
	DelayMs      int `json:"delay_ms"`
	DelaySeconds int `json:"delay_seconds"`
}

// Delay sleeps for a configurable duration and then succeeds.
type Delay struct{}

var _ maa.CustomActionRunner = &Delay{}

func (a *Delay) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := delayParam{}
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "Delay").Msg("failed to parse param")
		}
	}

	wait := time.Duration(param.DelayMs) * time.Millisecond
	if param.DelayMs <= 0 && param.DelaySeconds > 0 {
		wait = time.Duration(param.DelaySeconds) * time.Second
	}
	if wait <= 0 {
		return true
	}

	time.Sleep(wait)
	return true
}

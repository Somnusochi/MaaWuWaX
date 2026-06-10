// Package echofarm implements echo farm Custom Recognition for Wuthering Waves.
package echofarm

import (
	"fmt"
	"sync"
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// EchoOrbDetect — detects echo orbs (声骸球) on the ground for collection.
// Uses template matching with echo_orb.png.
// ---------------------------------------------------------------------------

type EchoOrbDetect struct{}

var _ maa.CustomRecognitionRunner = &EchoOrbDetect{}

func (r *EchoOrbDetect) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	detail, err := ctx.RunRecognition(
		"__EchoOrb_Detect",
		arg.Img,
		`{
			"__EchoOrb_Detect": {
				"recognition": "TemplateMatch",
				"template": "echo_orb.png",
				"threshold": 0.5
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		log.Debug().Str("component", "EchoOrbDetect").Msg("no echo orb found")
		return nil, false
	}

	log.Info().
		Str("component", "EchoOrbDetect").
		Int32("x", int32(detail.Box[0])).
		Int32("y", int32(detail.Box[1])).
		Msg("echo orb detected")

	return &maa.CustomRecognitionResult{
		Box:    detail.Box,
		Detail: detail.DetailJson,
	}, true
}

// ---------------------------------------------------------------------------
// FiveToOneMerge is registered by the echonhance package (full-featured version).
// ---------------------------------------------------------------------------

var echoFarmLoopState struct {
	sync.Mutex
	round int
}

type EchoFarmResetLoopAction struct{}

var _ maa.CustomActionRunner = &EchoFarmResetLoopAction{}

func (a *EchoFarmResetLoopAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	echoFarmLoopState.Lock()
	echoFarmLoopState.round = 0
	echoFarmLoopState.Unlock()
	log.Info().Str("component", "EchoFarmLoop").Msg("loop state reset")
	return true
}

type echoFarmLoopParam struct {
	MaxRounds int `json:"max_rounds"`
}

type EchoFarmNextRoundAction struct{}

var _ maa.CustomActionRunner = &EchoFarmNextRoundAction{}

func (a *EchoFarmNextRoundAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := echoFarmLoopParam{MaxRounds: 10000}
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "EchoFarmLoop").Msg("failed to parse param")
		}
	}
	if param.MaxRounds <= 0 {
		param.MaxRounds = 1
	}

	echoFarmLoopState.Lock()
	defer echoFarmLoopState.Unlock()

	if echoFarmLoopState.round >= param.MaxRounds {
		log.Info().
			Str("component", "EchoFarmLoop").
			Int("round", echoFarmLoopState.round).
			Int("max_rounds", param.MaxRounds).
			Msg("loop limit reached")
		return false
	}

	echoFarmLoopState.round++
	log.Info().
		Str("component", "EchoFarmLoop").
		Int("round", echoFarmLoopState.round).
		Int("max_rounds", param.MaxRounds).
		Msg("starting next farm round")
	return true
}

type echoFarmCollectParam struct {
	Method      string `json:"method"`
	MaxAttempts int    `json:"max_attempts"`
}

type EchoFarmCollectAction struct{}

var _ maa.CustomActionRunner = &EchoFarmCollectAction{}

func (a *EchoFarmCollectAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := echoFarmCollectParam{Method: "walk", MaxAttempts: 4}
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "EchoFarmCollect").Msg("failed to parse param")
		}
	}
	if param.MaxAttempts <= 0 {
		param.MaxAttempts = 4
	}

	if a.pick(ctx, param.MaxAttempts) {
		return true
	}

	switch param.Method {
	case "run_in_circle":
		a.pressFor(ctx, "W", 500*time.Millisecond)
		a.pressFor(ctx, "D", 1200*time.Millisecond)
		a.pressFor(ctx, "W", 600*time.Millisecond)
	case "yolo":
		a.pressFor(ctx, "A", 350*time.Millisecond)
		a.pressFor(ctx, "D", 700*time.Millisecond)
		a.pressFor(ctx, "A", 350*time.Millisecond)
	default:
		a.pressFor(ctx, "S", 350*time.Millisecond)
		a.pressFor(ctx, "W", 900*time.Millisecond)
	}

	return a.pick(ctx, param.MaxAttempts)
}

func (a *EchoFarmCollectAction) pick(ctx *maa.Context, maxAttempts int) bool {
	_, err := ctx.RunAction(
		"PickEnhanced",
		maa.Rect{0, 0, 1, 1},
		fmt.Sprintf(`{"max_attempts":%d}`, maxAttempts),
		nil,
	)
	return err == nil
}

func (a *EchoFarmCollectAction) pressFor(ctx *maa.Context, key string, duration time.Duration) {
	ctrl := ctx.GetTasker().GetController()
	code := keycode.MustCode(key)
	ctrl.PostKeyDown(code).Wait()
	time.Sleep(duration)
	ctrl.PostKeyUp(code).Wait()
	time.Sleep(120 * time.Millisecond)
}

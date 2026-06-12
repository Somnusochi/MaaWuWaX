// Package echofarm implements echo farm Custom Recognition for Wuthering Waves.
package echofarm

import (
	"strings"
	"sync"
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/mouse"
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
	detail, err := ctx.RunRecognition("EchoFarm_EchoOrbDetect", arg.Img)
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

var echoFarmRealmState struct {
	sync.Mutex
	param echoFarmEnterRealmParam
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
	Method string `json:"method"`
}

type EchoFarmCollectMoveAction struct{}

var _ maa.CustomActionRunner = &EchoFarmCollectMoveAction{}

func (a *EchoFarmCollectMoveAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := echoFarmCollectParam{Method: "walk"}
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "EchoFarmCollectMove").Msg("failed to parse param")
		}
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

	return true
}

func (a *EchoFarmCollectMoveAction) pressFor(ctx *maa.Context, key string, duration time.Duration) {
	ctrl := ctx.GetTasker().GetController()
	code := keycode.MustCode(key)
	ctrl.PostKeyDown(code).Wait()
	time.Sleep(duration)
	ctrl.PostKeyUp(code).Wait()
	time.Sleep(120 * time.Millisecond)
}

type echoFarmWalkUntilParam struct {
	Direction string `json:"direction"`
	StepMs    int    `json:"step_ms"`
	Focus     bool   `json:"focus"`
}

type EchoFarmPostTeleportWalkStepAction struct{}

var _ maa.CustomActionRunner = &EchoFarmPostTeleportWalkStepAction{}

func (a *EchoFarmPostTeleportWalkStepAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := echoFarmWalkUntilParam{Direction: "w", StepMs: 500, Focus: true}
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "EchoFarmPostTeleportWalkStep").Msg("failed to parse param")
		}
	}
	if param.Direction == "" {
		param.Direction = "w"
	}
	if param.StepMs <= 0 {
		param.StepMs = 500
	}

	ctrl := ctx.GetTasker().GetController()
	directionKey := strings.ToUpper(param.Direction)
	code, err := keycode.Code(directionKey)
	if err != nil {
		log.Warn().
			Err(err).
			Str("component", "EchoFarmPostTeleportWalkStep").
			Str("direction", param.Direction).
			Msg("invalid direction, falling back to W")
		directionKey = "W"
		code = keycode.MustCode(directionKey)
	}

	if param.Focus {
		mouse.MiddleClick(ctrl)
		ctrl.PostClickV2(640, 360, 1, 1).Wait()
		time.Sleep(100 * time.Millisecond)
	}

	echoFarmPressFor(ctrl, code, time.Duration(param.StepMs)*time.Millisecond)
	log.Debug().
		Str("component", "EchoFarmPostTeleportWalkStep").
		Str("direction", directionKey).
		Int("step_ms", param.StepMs).
		Msg("walked one post-teleport step")
	return true
}

type echoFarmEnterRealmParam struct {
	BossLevel    string `json:"boss_level"`
	BossProfile  string `json:"boss_profile"`
	CombatWaitMs int    `json:"combat_wait_ms"`
}

type EchoFarmEnterRealmFromFAction struct{}
type EchoFarmSelectRealmLevelAction struct{}
type EchoFarmAfterRealmEnterAction struct{}

var _ maa.CustomActionRunner = &EchoFarmEnterRealmFromFAction{}
var _ maa.CustomActionRunner = &EchoFarmSelectRealmLevelAction{}
var _ maa.CustomActionRunner = &EchoFarmAfterRealmEnterAction{}

func (a *EchoFarmEnterRealmFromFAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := parseEchoFarmEnterRealmParam(arg)
	saveEchoFarmEnterRealmParam(param)
	log.Info().
		Str("component", "EchoFarmEnterRealmFromF").
		Str("boss_level", param.BossLevel).
		Str("boss_profile", param.BossProfile).
		Int("combat_wait_ms", param.CombatWaitMs).
		Msg("prepared realm entry parameters")
	return true
}

func (a *EchoFarmSelectRealmLevelAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := loadEchoFarmEnterRealmParam()
	if arg != nil && arg.CustomActionParam != "" {
		param = parseEchoFarmEnterRealmParam(arg)
		saveEchoFarmEnterRealmParam(param)
	}
	return echoFarmSelectBossLevel(ctx, param.BossLevel)
}

func (a *EchoFarmAfterRealmEnterAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := loadEchoFarmEnterRealmParam()
	if arg != nil && arg.CustomActionParam != "" {
		param = parseEchoFarmEnterRealmParam(arg)
		saveEchoFarmEnterRealmParam(param)
	}
	echoFarmAfterRealmEnter(ctx, param)
	return true
}

func parseEchoFarmEnterRealmParam(arg *maa.CustomActionArg) echoFarmEnterRealmParam {
	param := echoFarmEnterRealmParam{BossLevel: "80"}
	if arg != nil && arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "EchoFarmEnterRealmParam").Msg("failed to parse param")
		}
	}
	if param.BossLevel == "" {
		param.BossLevel = "80"
	}
	return param
}

func saveEchoFarmEnterRealmParam(param echoFarmEnterRealmParam) {
	echoFarmRealmState.Lock()
	echoFarmRealmState.param = param
	echoFarmRealmState.Unlock()
}

func loadEchoFarmEnterRealmParam() echoFarmEnterRealmParam {
	echoFarmRealmState.Lock()
	defer echoFarmRealmState.Unlock()
	if echoFarmRealmState.param.BossLevel == "" {
		return echoFarmEnterRealmParam{BossLevel: "80"}
	}
	return echoFarmRealmState.param
}

func echoFarmInCombat(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition("EchoFarm_TeleportWalkCombat", nil)
	return err == nil && detail != nil && detail.Hit
}

func echoFarmHasFPrompt(ctx *maa.Context) bool {
	detail, err := ctx.RunRecognition("EchoFarm_TeleportWalkFPrompt", nil)
	return err == nil && detail != nil && detail.Hit
}

func echoFarmSelectBossLevel(ctx *maa.Context, level string) bool {
	if level == "" {
		level = "80"
	}
	detail, err := ctx.RunRecognition("EchoFarm_RealmLevelOCR", nil)
	if err != nil || detail == nil || !detail.Hit {
		log.Warn().Str("component", "EchoFarmEnterRealmFromF").Str("boss_level", level).Msg("boss level not found")
		return false
	}
	if !strings.Contains(detail.DetailJson, level) {
		log.Warn().
			Str("component", "EchoFarmEnterRealmFromF").
			Str("boss_level", level).
			Str("detail", detail.DetailJson).
			Msg("boss level OCR did not match")
		return false
	}
	ctx.GetTasker().GetController().PostClick(
		int32(detail.Box[0]+detail.Box[2]/2),
		int32(detail.Box[1]+detail.Box[3]/2),
	).Wait()
	time.Sleep(1000 * time.Millisecond)
	return true
}

func echoFarmAfterRealmEnter(ctx *maa.Context, param echoFarmEnterRealmParam) {
	profile := normalizeEchoFarmBossProfile(param.BossProfile)
	ctrl := ctx.GetTasker().GetController()

	switch profile {
	case "fallacy", "fallacyofnoreturn":
		echoFarmPressFor(ctrl, keycode.MustCode("D"), 250*time.Millisecond)
		echoFarmPressFor(ctrl, keycode.MustCode("W"), 700*time.Millisecond)
	case "fenrico":
		for i := 0; i < 3; i++ {
			if !echoFarmHasFPrompt(ctx) {
				break
			}
			ctrl.PostClickKey(keycode.MustCode("F")).Wait()
			time.Sleep(1000 * time.Millisecond)
		}
		echoFarmPressFor(ctrl, keycode.MustCode("W"), 1200*time.Millisecond)
	case "namelessexplorer":
		echoFarmPressFor(ctrl, keycode.MustCode("W"), 2500*time.Millisecond)
	case "sentryconstruct", "lionessofglory":
		time.Sleep(5 * time.Second)
	case "hyvatia":
		time.Sleep(7 * time.Second)
	}

	echoFarmWaitForBossProfile(param)
}

func echoFarmWaitForBossProfile(param echoFarmEnterRealmParam) {
	if param.CombatWaitMs > 0 {
		wait := time.Duration(param.CombatWaitMs) * time.Millisecond
		if param.CombatWaitMs <= 120 {
			wait = time.Duration(param.CombatWaitMs) * time.Second
		}
		time.Sleep(wait)
		return
	}
	switch normalizeEchoFarmBossProfile(param.BossProfile) {
	case "sentryconstruct", "lionessofglory", "fallacy", "fallacyofnoreturn":
		time.Sleep(5 * time.Second)
	case "hyvatia":
		time.Sleep(7 * time.Second)
	}
}

func echoFarmPressFor(ctrl *maa.Controller, code int32, duration time.Duration) {
	ctrl.PostKeyDown(code).Wait()
	time.Sleep(duration)
	ctrl.PostKeyUp(code).Wait()
}

func normalizeEchoFarmBossProfile(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, " ", "")
	text = strings.ReplaceAll(text, "\n", "")
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ReplaceAll(text, "：", ":")
	return strings.ToLower(text)
}

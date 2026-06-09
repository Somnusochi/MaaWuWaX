// Package navigation implements map navigation Custom Recognition for Wuthering Waves.
package navigation

import (
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/walk"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// MinimapNavigateRecognition — performs a single forward step (W key).
// Used by EchoFarm and other navigation flows.
// ---------------------------------------------------------------------------

type MinimapNavigateRecognition struct{}

var _ maa.CustomRecognitionRunner = &MinimapNavigateRecognition{}

func (r *MinimapNavigateRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	ctrl := ctx.GetTasker().GetController()
	walk.StepForward(ctrl)
	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"action":"step_forward"}`,
	}, true
}

// ---------------------------------------------------------------------------
// DirectionWalkAction — walks in a WASD direction for a specified duration.
// Param: {"direction": "forward"|"back"|"left"|"right", "duration_ms": 3000}
// ---------------------------------------------------------------------------

type DirectionWalkAction struct{}

var _ maa.CustomActionRunner = &DirectionWalkAction{}

type walkParam struct {
	Direction  string `json:"direction"`
	DurationMs int    `json:"duration_ms"`
	Sprint     bool   `json:"sprint"`
}

func (a *DirectionWalkAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := walkParam{
		Direction:  "forward",
		DurationMs: 2000,
		Sprint:     false,
	}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "DirectionWalk").Msg("failed to parse param")
		}
	}

	ctrl := ctx.GetTasker().GetController()
	dir := parseDirection(param.Direction)
	dur := time.Duration(param.DurationMs) * time.Millisecond

	log.Debug().
		Str("component", "DirectionWalk").
		Str("direction", param.Direction).
		Int("duration_ms", param.DurationMs).
		Bool("sprint", param.Sprint).
		Msg("walking")

	if param.Sprint {
		walk.Sprint(ctrl, dir, dur)
	} else {
		walk.Walk(ctrl, dir, dur)
	}

	return true
}

func parseDirection(s string) walk.Direction {
	switch s {
	case "back":
		return walk.Back
	case "left":
		return walk.Left
	case "right":
		return walk.Right
	default:
		return walk.Forward
	}
}

// ---------------------------------------------------------------------------
// TeleportBossAction — navigates F2 book to teleport to a boss.
// This is a simplified version; full boss selection would require OCR/index.
// ---------------------------------------------------------------------------

type TeleportBossAction struct{}

var _ maa.CustomActionRunner = &TeleportBossAction{}

func (a *TeleportBossAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	log.Info().Str("component", "TeleportBoss").Msg("teleporting to boss")
	ctrl := ctx.GetTasker().GetController()

	f2Code := keycode.MustCode("F2")
	escCode := keycode.MustCode("ESC")

	// Step 1: Open F2 book.
	ctrl.PostClickKey(f2Code).Wait()
	time.Sleep(2000 * time.Millisecond)

	// Step 2: Click boss tab.
	detail, err := ctx.RunRecognition(
		"__TeleportBoss_FindTab",
		nil,
		`{
			"__TeleportBoss_FindTab": {
				"recognition": "TemplateMatch",
				"template": "gray_book_boss.png",
				"threshold": 0.6
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		log.Warn().Str("component", "TeleportBoss").Msg("boss tab not found")
		ctrl.PostClickKey(escCode).Wait()
		return false
	}
	ctrl.PostClick(
		int32(detail.Box[0]+detail.Box[2]/2),
		int32(detail.Box[1]+detail.Box[3]/2),
	).Wait()
	time.Sleep(500 * time.Millisecond)

	// Step 3: Click proceed button.
	proceedDetail, err := ctx.RunRecognition(
		"__TeleportBoss_Proceed",
		nil,
		`{
			"__TeleportBoss_Proceed": {
				"recognition": "TemplateMatch",
				"template": "boss_proceed.png",
				"threshold": 0.7
			}
		}`,
	)
	if err != nil || proceedDetail == nil || !proceedDetail.Hit {
		log.Warn().Str("component", "TeleportBoss").Msg("proceed button not found")
		ctrl.PostClickKey(escCode).Wait()
		return false
	}
	ctrl.PostClick(
		int32(proceedDetail.Box[0]+proceedDetail.Box[2]/2),
		int32(proceedDetail.Box[1]+proceedDetail.Box[3]/2),
	).Wait()
	time.Sleep(1000 * time.Millisecond)

	// Step 4: Click travel button.
	travelDetail, err := ctx.RunRecognition(
		"__TeleportBoss_Travel",
		nil,
		`{
			"__TeleportBoss_Travel": {
				"recognition": "TemplateMatch",
				"template": "gray_teleport.png",
				"threshold": 0.7
			}
		}`,
	)
	if err != nil || travelDetail == nil || !travelDetail.Hit {
		log.Warn().Str("component", "TeleportBoss").Msg("travel button not found")
		ctrl.PostClickKey(escCode).Wait()
		return false
	}
	ctrl.PostClick(
		int32(travelDetail.Box[0]+travelDetail.Box[2]/2),
		int32(travelDetail.Box[1]+travelDetail.Box[3]/2),
	).Wait()
	time.Sleep(3000 * time.Millisecond)

	// Step 5: Wait for world load.
	for i := 0; i < 15; i++ {
		if ctx.GetTasker().Stopping() {
			return true
		}
		ctrl.PostScreencap().Wait()
		img, err := ctrl.CacheImage()
		if err != nil {
			time.Sleep(2000 * time.Millisecond)
			continue
		}
		worldDetail, err := ctx.RunRecognition(
			"__TeleportBoss_WaitWorld",
			img,
			`{
				"__TeleportBoss_WaitWorld": {
					"recognition": "TemplateMatch",
					"template": "minimap.png",
					"threshold": 0.7,
					"roi": [1050, 20, 200, 160]
				}
			}`,
		)
		if err == nil && worldDetail != nil && worldDetail.Hit {
			log.Info().Str("component", "TeleportBoss").Msg("arrived in world")
			return true
		}
		time.Sleep(2000 * time.Millisecond)
	}

	log.Warn().Str("component", "TeleportBoss").Msg("timeout waiting for world load")
	return false
}

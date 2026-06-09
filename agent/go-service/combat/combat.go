package combat

import (
	"encoding/json"
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

// ── CombatStateRecognition — ok-ww CombatCheck port ───────────────────

type CombatStateRecognition struct{}

var _ maa.CustomRecognitionRunner = &CombatStateRecognition{}

func (r *CombatStateRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	screenAnalyzer.Update(ctx, arg.Img)

	state := "fighting"
	if screenAnalyzer.HasDodge {
		state = "dodge"
	} else if !screenAnalyzer.InCombat() {
		if screenAnalyzer.PickupF {
			state = "loot"
		} else {
			state = "idle"
		}
	} else if screenAnalyzer.ConcertoPct >= 1.0 {
		state = "switch"
	}

	detail, _ := json.Marshal(map[string]any{
		"state":    state,
		"target":   screenAnalyzer.HasTarget,
		"hp":       screenAnalyzer.HasHPBar,
		"concerto": screenAnalyzer.ConcertoPct,
	})

	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1280, 720},
		Detail: string(detail),
	}, true
}

// ── CombatMainAction — state machine driven action ────────────────────

type CombatMainAction struct {
	phase       string
	lastDodge   time.Time
	lastAction  time.Time
	lastSwitch  time.Time
	rotationIdx int
}

func (a *CombatMainAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	if a.phase == "" {
		a.phase = "scan"
	}
	now := time.Now()
	roi := maa.Rect{0, 0, 1, 1}

	// Read state from param if available (set by previous CombatState recognition)
	if arg.CustomActionParam != "" {
		var s map[string]any
		json.Unmarshal([]byte(arg.CustomActionParam), &s)
	}

	switch a.phase {
	case "scan":
		if time.Since(a.lastAction) > 150*time.Millisecond {
			ctx.RunAction("Combat_DetectTarget", roi, "", nil)
			a.lastAction = now
		}

	case "rotation":
		if time.Since(a.lastAction) > 150*time.Millisecond {
			switch a.rotationIdx % 4 {
			case 0:
				ctx.RunAction("Combat_RotationLiberation", roi, "", nil)
			case 2:
				ctx.RunAction("Combat_RotationEcho", roi, "", nil)
			}
			if a.rotationIdx%2 == 1 {
				ctx.RunAction("Combat_RotationSkill1", roi, "", nil)
			}
			ctx.RunAction("Combat_RotationCombo", roi, "", nil)
			a.rotationIdx++
			a.lastAction = now
		}

	case "dodge":
		ctx.RunAction("Combat_Dodge", roi, "", nil)
		a.lastDodge = now
		time.Sleep(120 * time.Millisecond)
		a.phase = "rotation"

	case "loot":
		ctx.RunAction("Combat_LootAction", roi, "", nil)
		time.Sleep(250 * time.Millisecond)
		a.phase = "scan"
		log.Info().Str("component", "Combat").Msg("loot done")
	}

	return true
}

// ── CharacterDetectRecognition ────────────────────────────────────────

type CharacterDetectRecognition struct{}

var _ maa.CustomRecognitionRunner = &CharacterDetectRecognition{}

func (r *CharacterDetectRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	screenAnalyzer.Update(ctx, arg.Img)
	alive := []int{}
	for i, a := range screenAnalyzer.CharAlive {
		if a {
			alive = append(alive, i+1)
		}
	}
	d, _ := json.Marshal(map[string]any{"alive": alive})
	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1280, 720},
		Detail: string(d),
	}, true
}

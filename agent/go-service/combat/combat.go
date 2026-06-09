// Package combat implements auto-combat Custom Recognition and Action for Wuthering Waves.
package combat

import (
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
)

// ---------------------------------------------------------------------------
// CombatStateRecognition — detects dodge prompt, target lock, and idle state.
// Uses ScreenAnalyzer for batch detection from a single screenshot.
// ---------------------------------------------------------------------------

type CombatStateRecognition struct{}

var _ maa.CustomRecognitionRunner = &CombatStateRecognition{}

func (r *CombatStateRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	// Update the screen analyzer with the current frame.
	screenAnalyzer.Update(ctx, arg.Img)

	// Priority 1: Dodge prompt.
	if screenAnalyzer.HasDodge() {
		return &maa.CustomRecognitionResult{
			Box:    maa.Rect{500, 300, 280, 420},
			Detail: `{"state":"dodge"}`,
		}, true
	}

	// Priority 2: Target locked — actively fighting.
	if screenAnalyzer.HasTarget() {
		return &maa.CustomRecognitionResult{
			Box:    maa.Rect{400, 200, 800, 600},
			Detail: `{"state":"fighting"}`,
		}, true
	}

	// Priority 3: Pickup available.
	if screenAnalyzer.HasPickUp() {
		return &maa.CustomRecognitionResult{
			Box:    maa.Rect{300, 200, 680, 480},
			Detail: `{"state":"loot"}`,
		}, true
	}

	// Fallback: idle or dead.
	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1, 1},
		Detail: `{"state":"idle_or_dead"}`,
	}, true
}

// ---------------------------------------------------------------------------
// CombatMainAction — state-machine combat loop.
// Parses attach params, executes skill rotation with dodge and liberation.
// ---------------------------------------------------------------------------

type combatAttach struct {
	EnableDodge   bool   `json:"enable_dodge"`
	EnableLib     bool   `json:"enable_lib"`
	EnableConSwap bool   `json:"enable_con_swap"`
	ComboKey      string `json:"combo_key"`
	SkillKey1     string `json:"skill_key1"`
	SkillKey2     string `json:"skill_key2"`
	LibKey        string `json:"lib_key"`
	MaxCycles     int    `json:"max_cycles"`
}

func defaultAttach() combatAttach {
	return combatAttach{
		EnableDodge:   true,
		EnableLib:     true,
		EnableConSwap: false,
		ComboKey:      "Q",
		SkillKey1:     "E",
		SkillKey2:     "T",
		LibKey:        "R",
		MaxCycles:     0, // 0 = unlimited (Pipeline controls loop)
	}
}

type CombatMainAction struct{}

var _ maa.CustomActionRunner = &CombatMainAction{}

func (a *CombatMainAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	// Parse attach params with defaults.
	att := defaultAttach()
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &att); err != nil {
			log.Warn().Err(err).Str("component", "CombatMain").Msg("failed to parse attach, using defaults")
		}
	}

	log.Info().
		Str("component", "CombatMain").
		Bool("dodge", att.EnableDodge).
		Bool("lib", att.EnableLib).
		Str("combo", att.ComboKey).
		Msg("combat loop started")

	ctrl := ctx.GetTasker().GetController()

	tabCode := keycode.MustCode("TAB")
	wCode := keycode.MustCode("W")
	shiftCode := keycode.MustCode("SHIFT")

	comboCode, err := keycode.Code(att.ComboKey)
	if err != nil {
		comboCode = keycode.MustCode("Q")
	}
	skill1Code, err := keycode.Code(att.SkillKey1)
	if err != nil {
		skill1Code = keycode.MustCode("E")
	}
	skill2Code, err := keycode.Code(att.SkillKey2)
	if err != nil {
		skill2Code = keycode.MustCode("T")
	}
	libCode, err := keycode.Code(att.LibKey)
	if err != nil {
		libCode = keycode.MustCode("R")
	}

	// Engage: Tab to lock + W to approach.
	ctrl.PostClickKey(tabCode).Wait()
	time.Sleep(100 * time.Millisecond)
	ctrl.PostKeyDown(wCode).Wait()
	time.Sleep(500 * time.Millisecond)
	ctrl.PostKeyUp(wCode).Wait()
	time.Sleep(50 * time.Millisecond)

	// Main rotation loop.
	cycles := 0
	for {
		// Check stopping signal.
		if ctx.GetTasker().Stopping() {
			log.Info().Str("component", "CombatMain").Msg("stopping signal received")
			return true
		}

		// Capture and analyze screen.
		ctrl.PostScreencap().Wait()
		img, err := ctrl.CacheImage()
		if err != nil {
			log.Warn().Err(err).Str("component", "CombatMain").Msg("failed to capture screen")
			return false
		}
		screenAnalyzer.Update(ctx, img)

		// Exit condition: not in world and no target.
		if !screenAnalyzer.IsInWorld() && !screenAnalyzer.HasTarget() {
			// Check if dead — try to revive.
			if screenAnalyzer.IsDead() {
				log.Warn().Str("component", "CombatMain").Msg("character dead, attempting revive")
				reviveCode := keycode.MustCode("F")
				ctrl.PostClickKey(reviveCode).Wait()
				time.Sleep(3000 * time.Millisecond)
				continue
			}
			log.Info().Str("component", "CombatMain").Msg("exiting combat (not in world)")
			return true
		}

		// Dodge: Shift if dodge prompt detected.
		if att.EnableDodge && screenAnalyzer.HasDodge() {
			ctrl.PostClickKey(shiftCode).Wait()
			time.Sleep(200 * time.Millisecond)
			continue
		}

		// Liberation: R key if available and enabled.
		if att.EnableLib && screenAnalyzer.HasLiberation() {
			log.Debug().Str("component", "CombatMain").Msg("liberation available, pressing R")
			ctrl.PostClickKey(libCode).Wait()
			time.Sleep(1000 * time.Millisecond)
		}

		// Skill rotation: combo → skill1 → skill2 → combo.
		for _, key := range []int32{comboCode, skill1Code, skill2Code, comboCode} {
			if ctx.GetTasker().Stopping() {
				return true
			}
			ctrl.PostClickKey(key).Wait()
			time.Sleep(150 * time.Millisecond)
		}

		// Con swap: switch character when concerto is full.
		if att.EnableConSwap && screenAnalyzer.HasConFull() {
			log.Debug().Str("component", "CombatMain").Msg("concerto full, switching character")
			// Press 2 to switch to next character.
			switchCode := keycode.MustCode("2")
			ctrl.PostClickKey(switchCode).Wait()
			time.Sleep(500 * time.Millisecond)
		}

		// Re-engage: Tab + W if no target.
		if !screenAnalyzer.HasTarget() {
			ctrl.PostClickKey(tabCode).Wait()
			time.Sleep(100 * time.Millisecond)
			ctrl.PostKeyDown(wCode).Wait()
			time.Sleep(300 * time.Millisecond)
			ctrl.PostKeyUp(wCode).Wait()
			time.Sleep(50 * time.Millisecond)
		}

		cycles++
		if att.MaxCycles > 0 && cycles >= att.MaxCycles {
			log.Info().Int("cycles", cycles).Str("component", "CombatMain").Msg("max cycles reached")
			return true
		}
	}
}

// ---------------------------------------------------------------------------
// CharacterDetectRecognition — identifies current character by avatar template.
// ---------------------------------------------------------------------------

type CharacterDetectRecognition struct{}

var _ maa.CustomRecognitionRunner = &CharacterDetectRecognition{}

func (r *CharacterDetectRecognition) Run(ctx *maa.Context, arg *maa.CustomRecognitionArg) (*maa.CustomRecognitionResult, bool) {
	// Use DirectHit to run template match against all char_* templates.
	detail, err := ctx.RunRecognition(
		"__CharacterDetect",
		arg.Img,
		`{
			"__CharacterDetect": {
				"recognition": "DirectHit"
			}
		}`,
	)
	if err != nil || detail == nil || !detail.Hit {
		return nil, false
	}
	return &maa.CustomRecognitionResult{
		Box:    detail.Box,
		Detail: detail.DetailJson,
	}, true
}

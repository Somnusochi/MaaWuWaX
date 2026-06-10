package combat

import (
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/bytedance/sonic"
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

	detail, _ := sonic.Marshal(map[string]any{
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
	phase        string
	lastDodge    time.Time
	lastAction   time.Time
	lastSwitch   time.Time
	lastSwitchIn [3]time.Time
	rotationIdx  int
	switchIdx    int
	charStates   map[string]*combatCharState
}

type combatMainParam struct {
	UseLiberation bool `json:"use_liberation"`
	AutoTarget    bool `json:"auto_target"`
}

func (a *CombatMainAction) Run(ctx *maa.Context, arg *maa.CustomActionArg) bool {
	param := combatMainParam{UseLiberation: true, AutoTarget: true}
	if arg.CustomActionParam != "" {
		if err := sonic.Unmarshal([]byte(arg.CustomActionParam), &param); err != nil {
			log.Warn().Err(err).Str("component", "Combat").Msg("failed to parse param")
		}
	}

	now := time.Now()
	roi := maa.Rect{0, 0, 1, 1}

	ctrl := ctx.GetTasker().GetController()
	ctrl.PostScreencap().Wait()

	// Track frame for freeze compensation (ok-ww: time_elapsed_accounting_for_freeze)
	screenAnalyzer.TrackFrame(now.UnixNano())

	img, err := ctrl.CacheImage()
	if err == nil && img != nil {
		screenAnalyzer.Update(ctx, img)
	}

	// ── Dodge ──
	if screenAnalyzer.HasDodge && time.Since(a.lastDodge) > 500*time.Millisecond {
		ctx.RunAction("Combat_Dodge", roi, "", nil)
		a.lastDodge = now
		time.Sleep(120 * time.Millisecond)
		return true
	}

	// ── Out of combat ──
	if !screenAnalyzer.InCombat() {
		if screenAnalyzer.PickupF {
			ctx.RunAction("Combat_LootAction", roi, "", nil)
			time.Sleep(250 * time.Millisecond)
			return true
		}
		if param.AutoTarget && !screenAnalyzer.HasTarget {
			// Auto-target: middle-click to lock on to nearest enemy (ok-ww: target_enemy)
			ctrl.PostClick(640, 360).Wait()
			time.Sleep(200 * time.Millisecond)
		}
		time.Sleep(120 * time.Millisecond)
		return true
	}

	// ── Health-based switch: if current char is low HP, switch to healer ──
	if screenAnalyzer.HasHealthLow() && time.Since(a.lastSwitch) > 3*time.Second {
		for i, slot := range screenAnalyzer.CharSlots {
			if slot.Role == roleHealer && slot.Alive && i != screenAnalyzer.CurrentIdx {
				a.performCharacterStrategy(ctx, param) // quick perform then switch
				a.lastSwitch = now
				time.Sleep(120 * time.Millisecond)
				return true
			}
		}
	}

	// ── Concerto switch ──
	if time.Since(a.lastSwitch) > 8*time.Second && screenAnalyzer.ConcertoPct >= 1.0 {
		target := a.chooseSwitchTarget(now)
		if target >= 0 {
			ctx.RunAction(switchActionName(target), roi, "", nil)
			a.lastSwitchIn[target] = now
			a.switchIdx++
			a.lastSwitch = now
			a.rotationIdx = 0
			time.Sleep(300 * time.Millisecond)
			return true
		}
	}

	if time.Since(a.lastAction) <= 120*time.Millisecond {
		return true
	}

	a.performCharacterStrategy(ctx, param)
	a.rotationIdx++
	a.lastAction = now

	if screenAnalyzer.PickupF {
		ctx.RunAction("Combat_LootAction", roi, "", nil)
		time.Sleep(250 * time.Millisecond)
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
	d, _ := sonic.Marshal(map[string]any{
		"alive":       alive,
		"current":     screenAnalyzer.CurrentIdx + 1,
		"team_size":   screenAnalyzer.TeamSize,
		"char_slots":  screenAnalyzer.CharSlots,
		"concerto":    screenAnalyzer.ConcertoPct,
		"liberation":  screenAnalyzer.Liberation,
		"in_combat":   screenAnalyzer.InCombat(),
		"has_target":  screenAnalyzer.HasTarget,
		"has_hp_bar":  screenAnalyzer.HasHPBar,
		"has_boss_hp": screenAnalyzer.HasBossHP,
	})
	return &maa.CustomRecognitionResult{
		Box:    maa.Rect{0, 0, 1280, 720},
		Detail: string(d),
	}, true
}

func (a *CombatMainAction) chooseSwitchTarget(now time.Time) int {
	current := screenAnalyzer.CurrentIdx
	if current < 0 || current >= 3 {
		return a.fallbackSwitchTarget()
	}

	candidates := make([]int, 0, 2)
	for i, slot := range screenAnalyzer.CharSlots {
		if i == current || !slot.Alive {
			continue
		}
		candidates = append(candidates, i)
	}
	if len(candidates) == 0 {
		return -1
	}

	currentRole := screenAnalyzer.CharSlots[current].Role
	if currentRole == roleMain {
		if target := a.oldestByRole(candidates, roleSub, now, true); target >= 0 {
			return target
		}
		if target := a.oldestByRole(candidates, roleHealer, now, true); target >= 0 {
			return target
		}
	}

	if currentRole == roleSub || currentRole == roleHealer {
		if target := a.oldestByRole(candidates, roleMain, now, false); target >= 0 {
			return target
		}
	}

	return a.oldest(candidates)
}

func (a *CombatMainAction) fallbackSwitchTarget() int {
	for step := 0; step < 3; step++ {
		idx := (a.switchIdx + step + 1) % 3
		if screenAnalyzer.CharAlive[idx] {
			return idx
		}
	}
	return -1
}

func (a *CombatMainAction) oldestByRole(candidates []int, role charRole, now time.Time, preferExpiredBuff bool) int {
	filtered := make([]int, 0, len(candidates))
	for _, idx := range candidates {
		slot := screenAnalyzer.CharSlots[idx]
		if slot.Role == role {
			filtered = append(filtered, idx)
		}
	}
	if len(filtered) == 0 {
		return -1
	}
	if preferExpiredBuff {
		expired := make([]int, 0, len(filtered))
		for _, idx := range filtered {
			meta := screenAnalyzer.CharSlots[idx]
			if meta.Role == roleUnknown || now.Sub(a.lastSwitchIn[idx]).Seconds() >= charBuffTime(meta.Name) {
				expired = append(expired, idx)
			}
		}
		if len(expired) > 0 {
			filtered = expired
		}
	}
	return a.oldest(filtered)
}

func (a *CombatMainAction) oldest(candidates []int) int {
	best := candidates[0]
	bestTime := a.lastSwitchIn[best]
	for _, idx := range candidates[1:] {
		if a.lastSwitchIn[idx].Before(bestTime) {
			best = idx
			bestTime = a.lastSwitchIn[idx]
		}
	}
	return best
}

func charBuffTime(name string) float64 {
	for _, meta := range charTemplates {
		if meta.Name == name {
			if meta.BuffTime <= 0 {
				return 0
			}
			return meta.BuffTime
		}
	}
	return 0
}

func switchActionName(index int) string {
	return []string{"Combat_SwitchChar1", "Combat_SwitchChar2", "Combat_SwitchChar3"}[index]
}

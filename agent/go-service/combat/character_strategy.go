package combat

import (
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

type combatCharState struct {
	lastPerform    time.Time
	lastEcho       time.Time
	lastHeavy      time.Time
	lastLiberation time.Time
	lastResonance  time.Time
	lastBuff       time.Time     // support buff tracking (Chisa)
	introTime      time.Duration // Aemeath intro timer
	flag           bool
	flag2          bool
	phaseUntil     time.Time
}

type combatActor struct {
	action *CombatMainAction
	ctx    *maa.Context
	param  combatMainParam
	state  *combatCharState
	slot   charSlot
	roi    maa.Rect
}

func (a *CombatMainAction) performCharacterStrategy(ctx *maa.Context, param combatMainParam) {
	if a.charStates == nil {
		a.charStates = map[string]*combatCharState{}
	}
	slot := a.currentSlot()
	key := slot.Name
	if key == "" {
		key = "unknown"
	}
	state := a.charStates[key]
	if state == nil {
		state = &combatCharState{}
		a.charStates[key] = state
	}

	actor := combatActor{
		action: a,
		ctx:    ctx,
		param:  param,
		state:  state,
		slot:   slot,
		roi:    maa.Rect{0, 0, 1, 1},
	}

	log.Debug().Str("component", "Combat").Str("char", key).Str("role", string(slot.Role)).Msg("perform character strategy")

	// Dispatch to per-character strategy file in char/ package.
	dispatchCharStrategy(actor)

	state.lastPerform = time.Now()
}

func (a *CombatMainAction) currentSlot() charSlot {
	if screenAnalyzer.CurrentIdx >= 0 && screenAnalyzer.CurrentIdx < len(screenAnalyzer.CharSlots) {
		return screenAnalyzer.CharSlots[screenAnalyzer.CurrentIdx]
	}
	for _, slot := range screenAnalyzer.CharSlots {
		if slot.Current {
			return slot
		}
	}
	return charSlot{Name: "unknown", Role: roleUnknown, Alive: true}
}

// combat primitives shared across character files.

func (c combatActor) attack() bool {
	c.run("Combat_RotationCombo")
	return true
}

func (c combatActor) attackFor(duration time.Duration) {
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		c.attack()
		c.sleep(90 * time.Millisecond)
	}
}

func (c combatActor) skill() bool {
	if time.Since(c.state.lastResonance) < 2*time.Second {
		return false
	}
	c.run("Combat_RotationSkill1")
	c.state.lastResonance = time.Now()
	return true
}

func (c combatActor) liberation() bool {
	if !c.param.UseLiberation {
		return false
	}
	if time.Since(c.state.lastLiberation) < 12*time.Second {
		return false
	}
	if !screenAnalyzer.Liberation && !c.state.lastLiberation.IsZero() {
		return false
	}
	c.run("Combat_RotationLiberation")
	c.state.lastLiberation = time.Now()
	return true
}

func (c combatActor) echo() bool {
	if time.Since(c.state.lastEcho) < 18*time.Second {
		return false
	}
	c.run("Combat_RotationEcho")
	c.state.lastEcho = time.Now()
	return true
}

func (c combatActor) heavy(duration time.Duration) bool {
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostKeyDown(keycode.MustCode("Q")).Wait()
	c.sleep(duration)
	ctrl.PostKeyUp(keycode.MustCode("Q")).Wait()
	return true
}

func (c combatActor) forwardAttackFor(duration time.Duration) {
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostKeyDown(keycode.MustCode("W")).Wait()
	c.attackFor(duration)
	ctrl.PostKeyUp(keycode.MustCode("W")).Wait()
}

func (c combatActor) jumpAttackFor(duration time.Duration) {
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostClickKey(keycode.MustCode("SPACE")).Wait()
	c.sleep(120 * time.Millisecond)
	c.attackFor(duration)
}

func (c combatActor) requestSwitch() {
	target := c.action.chooseSwitchTarget(time.Now())
	if target < 0 {
		return
	}
	c.run(switchActionName(target))
	c.action.lastSwitchIn[target] = time.Now()
	c.action.lastSwitch = time.Now()
}

func (c combatActor) run(action string) {
	c.ctx.RunAction(action, c.roi, "", nil)
}

func (c combatActor) sleep(duration time.Duration) {
	if duration <= 0 {
		return
	}
	time.Sleep(duration)
}

func (c combatActor) recentlySwitchedIn(window time.Duration) bool {
	if window <= 0 {
		window = 1500 * time.Millisecond
	}
	idx := c.slot.Index
	if idx < 0 || idx >= len(c.action.lastSwitchIn) {
		return c.state.lastPerform.IsZero()
	}
	ts := c.action.lastSwitchIn[idx]
	if ts.IsZero() {
		return c.state.lastPerform.IsZero()
	}
	return time.Since(ts) <= window
}

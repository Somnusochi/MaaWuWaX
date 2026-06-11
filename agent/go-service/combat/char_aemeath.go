package combat

import "time"

// performAemeath mirrors ok-ww Aemeath.do_perform():
//
//	intro(1.2s attack, record_intro_lib, set per-outro intro_time) → perform_everything loop:
//	  handle_heavy → intro_lib1/lib2 cast → enhance_e(resonance+echo) → lib fallthrough → normal attack → switch
func performAemeath(c combatActor) {
	if c.recentlyIntroSwitchedIn(1700 * time.Millisecond) {
		c.state.lastIntro = time.Now()
		c.state.aemeathIntroFreeze = screenAnalyzer.FreezeDuration
		// Per-character intro_time from Python: 14s for Linnai/Lupa, 10s for Changli, -1 disabled for others.
		c.state.aemeathIntroTime = -1
		if c.switchedFromAny(2*time.Second, "linnai", "linnai2") || c.switchedFromAny(2*time.Second, "lupa", "lupa2") {
			c.state.aemeathIntroTime = 14
		}
		if c.switchedFromAny(2*time.Second, "changli", "changli2") {
			c.state.aemeathIntroTime = 10
		}

		c.attackFor(1200 * time.Millisecond)
	}

	start := time.Now()
	startFreeze := screenAnalyzer.FreezeDuration
	shouldWait := aemeathShouldWaitLib2(c) || aemeathShouldWaitEnhanceE(c)
	if !shouldWait && c.recentlyIntroSwitchedIn(1700*time.Millisecond) {
		shouldWait = aemeathLiberationCooldownLeft(c) < 12*time.Second
	}
	for {
		elapsed := c.freezeElapsed(start, startFreeze)
		if !(elapsed < 1200*time.Millisecond || (shouldWait && elapsed < 3600*time.Millisecond)) {
			break
		}
		if aemeathHandleHeavy(c) {
			c.fBreak()
			start = time.Now()
			startFreeze = screenAnalyzer.FreezeDuration
			shouldWait = aemeathShouldWaitLib2(c)
			continue
		}
		if aemeathIntroLibReady(c) && aemeathCastLiberation(c) {
			start = time.Now()
			startFreeze = screenAnalyzer.FreezeDuration
			shouldWait = aemeathShouldWaitLib2(c)
			continue
		}
		enhanceEReady := c.aemeathEnhanceEReady()
		if enhanceEReady {
			if aemeathClickEnhanceE(c) {
				c.state.lastEnhanceE = time.Now()
				c.state.aemeathEnhanceEFreeze = screenAnalyzer.FreezeDuration
				c.echo()
				c.fBreak()
			}
			if (aemeathIntroLibReady(c) && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05)) || c.hasLongAction() {
				start = time.Now()
				startFreeze = screenAnalyzer.FreezeDuration
			} else {
				c.attack()
				c.sleep(10 * time.Millisecond)
				return
			}
		} else if aemeathCastLiberation(c) {
			start = time.Now()
			startFreeze = screenAnalyzer.FreezeDuration
			shouldWait = aemeathShouldWaitLib2(c)
			continue
		} else {
			c.attack()
		}
		c.sleep(90 * time.Millisecond)
	}
	c.requestSwitch()
}

// aemeathClickEnhanceE mirrors ok-ww Aemeath.enhance_e_available() + click_resonance():
// casts enhanced E (resonance with animation) for up to 1.5s.
func aemeathClickEnhanceE(c combatActor) bool {
	start := time.Now()
	startFreeze := screenAnalyzer.FreezeDuration
	clicked := false
	for c.currentResonance() > 0.05 && c.freezeElapsed(start, startFreeze) < 1500*time.Millisecond {
		if c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

// aemeathIntroLibReady mirrors ok-ww Aemeath.intro_lib1_ready():
// true when intro liberation window (14s) is still active.
func aemeathIntroLibReady(c combatActor) bool {
	if c.state.lastIntro.IsZero() {
		return false
	}
	// intro_time of -1 means disabled (no outro from Linnai/Lupa/Changli).
	if c.state.aemeathIntroTime <= 0 {
		return false
	}
	return c.freezeElapsed(c.state.lastIntro, c.state.aemeathIntroFreeze) <= time.Duration(c.state.aemeathIntroTime)*time.Second
}

// aemeathShouldWaitLib2 mirrors ok-ww Aemeath.should_wait_for_lib2():
// true when pending_lib2 or inside the LIB2_PREPARE_WINDOW (17s-25s after last_liber).
func aemeathShouldWaitLib2(c combatActor) bool {
	if c.state.pendingLiberation2 {
		return true
	}
	anchor, freezeAt, ok := aemeathLib2Anchor(c)
	if !ok {
		return false
	}
	elapsed := c.freezeElapsed(anchor, freezeAt)
	return elapsed > 17*time.Second && elapsed < 25*time.Second
}

// aemeathShouldWaitEnhanceE mirrors ok-ww Aemeath.should_wait_for_enhance_e():
// true when 12s have passed since last_enhance_e.
func aemeathShouldWaitEnhanceE(c combatActor) bool {
	if c.state.lastEnhanceE.IsZero() {
		return true
	}
	return c.freezeElapsed(c.state.lastEnhanceE, c.state.aemeathEnhanceEFreeze) > 12*time.Second
}

// aemeathHandleHeavy mirrors ok-ww Aemeath.handle_heavy():
// holds heavy attack while long_action is visible; marks pending_lib2 if inside the lib2 prepare window.
func aemeathHandleHeavy(c combatActor) bool {
	if !c.hasLongAction() {
		return false
	}
	preparesLib2 := aemeathShouldWaitLib2(c)
	c.holdHeavyUntil(1200*time.Millisecond, 100*time.Millisecond, func() bool {
		return !c.hasLongAction()
	})
	if preparesLib2 {
		c.state.pendingLiberation2 = true
	}
	return true
}

// aemeathCastLiberation mirrors ok-ww Aemeath.lib():
// casts lib1 or lib2 depending on lib2_available; records state accordingly.
func aemeathCastLiberation(c combatActor) bool {
	lib2 := c.aemeathLib2Ready()
	if !lib2 && !aemeathCanCastLib1(c) {
		return false
	}
	if !c.liberation() {
		return false
	}
	if lib2 {
		c.state.pendingLiberation2 = false
		c.state.lastLiberation = time.Now()
		c.state.aemeathLiberationFreeze = screenAnalyzer.FreezeDuration
		c.state.lastEnhanceE = c.state.lastLiberation
		c.state.aemeathEnhanceEFreeze = c.state.aemeathLiberationFreeze
	} else {
		c.state.lastIntro = time.Time{}
		c.state.aemeathIntroFreeze = 0
	}
	c.fBreak()
	return true
}

// aemeathLib2Anchor mirrors ok-ww Aemeath.lib2_cooldown_anchor():
// returns last_liberation time (or combat_start if never used) as the lib2 cooldown anchor.
func aemeathLib2Anchor(c combatActor) (time.Time, int64, bool) {
	if !c.state.lastLiberation.IsZero() {
		return c.state.lastLiberation, c.state.aemeathLiberationFreeze, true
	}
	if c.action == nil || c.action.combatStart.IsZero() {
		return time.Time{}, 0, false
	}
	return c.action.combatStart, 0, true
}

// aemeathLiberationCooldownLeft mirrors ok-ww Aemeath.liberation_cooldown_left():
// returns remaining time of the 25s liberation cooldown.
func aemeathLiberationCooldownLeft(c combatActor) time.Duration {
	if c.state.lastLiberation.IsZero() {
		return 0
	}
	elapsed := c.freezeElapsed(c.state.lastLiberation, c.state.aemeathLiberationFreeze)
	left := 25*time.Second - elapsed
	if left < 0 {
		return 0
	}
	return left
}

// aemeathCanCastLib1 mirrors ok-ww Aemeath.can_cast_lib1():
// true when cooldown is clear AND lib1_unlocked (intro window or 30s anchor).
func aemeathCanCastLib1(c combatActor) bool {
	return aemeathLiberationCooldownLeft(c) <= 0 && aemeathLib1Unlocked(c)
}

// aemeathLib1Unlocked mirrors ok-ww Aemeath.lib1_unlocked():
// unlocked if intro_lib1_ready or >=30s elapsed since lib anchor.
func aemeathLib1Unlocked(c combatActor) bool {
	if aemeathIntroLibReady(c) {
		return true
	}
	anchor, freezeAt, ok := aemeathLib2Anchor(c)
	if !ok {
		return false
	}
	return c.freezeElapsed(anchor, freezeAt) >= 30*time.Second
}

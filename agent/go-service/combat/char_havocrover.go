package combat

import "time"

// performHavocRover mirrors ok-ww HavocRover.do_perform():
//
//	element dispatch: Havoc→heavy_click_forte+liberation+resonance /
//	Spectro→intro→heavy→normal→echo→forte+resonance→liberation /
//	Wind→intro_flying→click_while_flying→liberation→wait_down /
//	  resonance→echo→resonance_spam→flying_click→liberation /
//	Basic→intro→wait_down→echo→liberation+resonance→normal
//	fast_perform_wind bypass when wind+need_fast
func performHavocRover(c combatActor) {
	if !c.recentlyIntroSwitchedIn(1600 * time.Millisecond) {
		c.sleep(10 * time.Millisecond)
	}

	if c.ringElement() == ringElementWind && c.needFastPerform() {
		havocRoverFastPerformWind(c)
		c.requestSwitch()
		return
	}

	switch c.ringElement() {
	case ringElementHavoc:
		havocRoverPerformHavoc(c)
	case ringElementSpectro:
		havocRoverPerformSpectro(c)
	case ringElementWind:
		havocRoverPerformWind(c)
	default:
		havocRoverPerformBasic(c)
	}
	c.requestSwitch()
}

// havocRoverFastPerformWind mirrors ok-ww HavocRover.fast_perform_wind_routine():
// condensed wind routine: intro_fly_click(0.5s) → flying→liberation+wait_down → forte→resonance → echo → resonance → attack.
func havocRoverFastPerformWind(c combatActor) {
	if c.recentlyIntroSwitchedIn(1600 * time.Millisecond) {
		if havocRoverClickWhileFlying(c, 500*time.Millisecond) {
			return
		}
	}
	if havocRoverWindFlying(c) {
		havocRoverClickLiberation(c)
		havocRoverWindWaitDown(c, false)
		c.sleep(30 * time.Millisecond)
	}
	if c.forteFull() {
		c.forceSkill()
		return
	}
	c.echo()
	if c.currentResonance() > 0.05 && !havocRoverWindFlying(c) {
		c.forceSkill()
		c.sleep(100 * time.Millisecond)
	}
	attackFor := 1*time.Second - c.performElapsed()
	if attackFor > 0 && havocRoverWindFlying(c) {
		havocRoverClickWhileFlying(c, attackFor)
	}
	if c.teamHas("cartethyia") && c.teamHas("phoebe") {
		havocRoverClickResonance(c)
	}
	if havocRoverClickLiberation(c) {
		c.sleep(30 * time.Millisecond)
	}
	if c.forteFull() {
		c.forceSkill()
	}
}

// havocRoverPerformBasic mirrors ok-ww HavocRover.perform_basic_routine():
// intro→attack→wait_down→echo→liberation→resonance→!liber+!res→normal_attack(1s).
func havocRoverPerformBasic(c combatActor) {
	if c.recentlyIntroSwitchedIn(1600 * time.Millisecond) {
		c.attackFor(1100 * time.Millisecond)
	}
	c.waitDown(1200 * time.Millisecond)
	c.echoWait(1 * time.Second)
	liber := havocRoverClickLiberation(c)
	res := havocRoverClickResonance(c)
	if !(liber || res) {
		c.attackFor(1 * time.Second)
	}
}

// havocRoverPerformHavoc mirrors ok-ww HavocRover.perform_havoc_routine():
// wait_down→heavy_click_forte→liberation→resonance→echo→remaining_time_attack.
func havocRoverPerformHavoc(c combatActor) {
	c.waitDown(1200 * time.Millisecond)
	if c.mouseForteFull() {
		c.holdHeavyUntil(1200*time.Millisecond, 100*time.Millisecond, func() bool {
			return !c.mouseForteFull()
		})
	}
	havocRoverClickLiberation(c)
	if havocRoverClickResonance(c) {
		return
	}
	c.echoWait(1 * time.Second)
	if remaining := 1100*time.Millisecond - c.performElapsed(); remaining > 0 {
		c.attackFor(remaining)
	}
}

// havocRoverPerformSpectro mirrors ok-ww HavocRover.perform_spectro_routine():
// intro(attack 1s)→wait_down→heavy→sleep(0.4s)→normal(0.7s)→echo→forte+resonance→normal(1.4s)→liberation/resonance.
func havocRoverPerformSpectro(c combatActor) {
	if c.recentlyIntroSwitchedIn(1600 * time.Millisecond) {
		c.attackFor(1 * time.Second)
	}
	c.waitDown(1200 * time.Millisecond)
	c.heavy(600 * time.Millisecond)
	c.sleep(400 * time.Millisecond)
	c.attackFor(700 * time.Millisecond)
	c.echo()
	if c.forteFull() && havocRoverClickResonance(c) {
		c.attackFor(1400 * time.Millisecond)
		c.sleep(100 * time.Millisecond)
	}
	if !havocRoverClickLiberation(c) {
		havocRoverClickResonance(c)
	}
}

// havocRoverPerformWind mirrors ok-ww HavocRover.perform_wind_routine():
// intro_flying→click_while_flying(2s)→liberation→wait_down /
//   wait_down→resonance+echo→resonance_spam(1s)→flying_click→liberation→wait_down.
func havocRoverPerformWind(c combatActor) {
	if c.recentlyIntroSwitchedIn(1600*time.Millisecond) && c.flying() {
		havocRoverClickWhileFlying(c, 2*time.Second)
		if havocRoverClickLiberation(c) {
			havocRoverWindWaitDown(c, false)
			return
		}
	}
	havocRoverWindWaitDown(c, false)
	if c.currentResonance() > 0.05 && !c.forteFull() {
		c.echo()
		start := time.Now()
		flying := false
		for time.Since(start) < 1*time.Second {
			c.forceSkill()
			c.attack()
			c.sleep(100 * time.Millisecond)
			if havocRoverWindFlying(c) {
				flying = true
				break
			}
		}
		if flying {
			if c.teamHas("cartethyia") && c.teamHas("phoebe") {
				havocRoverClickWhileFlying(c, 1600*time.Millisecond)
				if havocRoverClickResonance(c) {
					havocRoverClickWhileFlying(c, 1*time.Second)
				}
			} else {
				havocRoverClickWhileFlying(c, 1740*time.Millisecond)
			}
		}
	}
	havocRoverClickLiberation(c)
	havocRoverWindWaitDown(c, true)
}

// KNOWN_DIFF: Python's wind_routine_flying checks has_lavitator first,
// using self.flying() only when lavitator is present; Go has no HasLavitator
// member so we fall back to the unconditional flying check.
func havocRoverWindFlying(c combatActor) bool {
	if c.flying() {
		return true
	}
	return c.currentResonance() > 0.15
}

func havocRoverClickWhileFlying(c combatActor, duration time.Duration) bool {
	start := time.Now()
	for time.Since(start) < duration {
		if !havocRoverWindFlying(c) {
			return false
		}
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	return true
}

// KNOWN_DIFF: Python's wind_routine_wait_down checks has_lavitator and
// either calls self.wait_down() (lavitator) or waits for resonance<0.15
// (no lavitator). Go uses c.flying() as the lavitator proxy.
func havocRoverWindWaitDown(c combatActor, checkForte bool) {
	if c.flying() {
		// has_lavitator path: wait until not flying
		c.waitDown(2500 * time.Millisecond)
	} else if c.currentResonance() > 0.15 {
		// no-lavitator path: wait for resonance to drop below threshold
		deadline := time.Now().Add(2500 * time.Millisecond)
		for time.Now().Before(deadline) && c.currentResonance() > 0.15 {
			c.sleep(100 * time.Millisecond)
		}
	}
	if checkForte {
		c.sleep(30 * time.Millisecond)
		if c.forteFull() {
			c.forceSkill()
		}
	} else {
		c.sleep(10 * time.Millisecond)
	}
}

// havocRoverClickLiberation mirrors ok-ww HavocRover.click_liberation(send_click=True):
// standard liberation cast with finishLiberationCast.
func havocRoverClickLiberation(c combatActor) bool {
	if !c.param.UseLiberation || (!screenAnalyzer.Liberation && c.currentLiberation() <= 0.05) {
		return false
	}
	start := time.Now()
	clicked := false
	for time.Since(start) < 800*time.Millisecond && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) {
		c.forceLiberation()
		clicked = true
		c.sleep(100 * time.Millisecond)
	}
	return finishLiberationCast(c, clicked, 3*time.Second)
}

// havocRoverClickResonance mirrors ok-ww HavocRover.click_resonance(send_click=True):
// casts resonance while available for up to 15s.
func havocRoverClickResonance(c combatActor) bool {
	if c.currentResonance() <= 0.05 {
		return false
	}
	start := time.Now()
	clicked := false
	for c.currentResonance() > 0.05 && time.Since(start) < 15*time.Second {
		if c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

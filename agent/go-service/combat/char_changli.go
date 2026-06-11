package combat

import "time"

// performChangli mirrors ok-ww Changli.do_perform():
//
//	intro(0.3s attack → enhanced normal=true) → judge_forte →
//	enhanced: 0.2s attack + 0.2s sleep → brant outro→do_perform_outro →
//	forte>=4/mouse_forte_full: heavy_click_forte → switch /
//	forte<3+liberation: liberation_and_heavy(hold heavy during animation) → switch /
//	flick_resonance(set enhanced) → switch / echo → normal attack → switch
func performChangli(c combatActor) {
	forte := -1
	if c.recentlyIntroSwitchedIn(1400 * time.Millisecond) {
		c.attackFor(300 * time.Millisecond)
		c.state.changliEnhancedNormal = true
	}

	forte = c.changliForteTier()
	if c.state.changliEnhancedNormal {
		c.attackFor(200 * time.Millisecond)
		c.sleep(200 * time.Millisecond)
		if c.switchedFromName("brant", 1700*time.Millisecond) {
			c.sleep(200 * time.Millisecond)
			changliPerformOutro(c, c.changliForteTier())
			c.requestSwitch()
			return
		}
		if forte == 3 {
			c.sleep(200 * time.Millisecond)
			forte = c.changliForteTier()
		}
	}
	c.state.changliEnhancedNormal = false

	if forte >= 4 || c.mouseForteFull() {
		changliHeavyRelease(c)
		c.requestSwitch()
		return
	}

	if !(forte >= 3 && c.resonanceAvailable()) && changliLiberationAndHeavy(c, false, 0, 5*time.Second) {
		c.requestSwitch()
		return
	}

	if changliFlickResonance(c, 200*time.Millisecond, false) {
		c.state.changliEnhancedNormal = true
		c.requestSwitch()
		return
	}

	if c.echoWait(1 * time.Second) {
		c.requestSwitch()
		return
	}

	c.attackFor(100 * time.Millisecond)
	c.requestSwitch()
}

// changliPerformOutro mirrors ok-ww Changli.do_perform_outro():
// during brant outro: forte=3→resonance+attack loop to build forte / forte>=4→heavy_release → liberation_and_heavy → flick_resonance → echo.
func changliPerformOutro(c combatActor, forte int) {
	if forte == 3 {
		start := time.Now()
		forteReadyAt := start
		resPending := true
		for time.Since(start) < 5*time.Second {
			if resPending && changliFlickResonance(c, 200*time.Millisecond, false) {
				resPending = false
				continue
			}
			c.attack()
			if c.mouseForteFull() {
				if time.Since(forteReadyAt) > 200*time.Millisecond {
					break
				}
			} else {
				forteReadyAt = time.Now()
			}
			c.sleep(100 * time.Millisecond)
		}
		if c.mouseForteFull() {
			changliHeavyRelease(c)
			c.sleep(1 * time.Second)
		}
	} else if forte >= 4 || c.mouseForteFull() {
		changliHeavyRelease(c)
		forte = 0
		c.sleep(1 * time.Second)
	}

	if c.liberationAvailable() && changliLiberationAndHeavy(c, false, 0, 5*time.Second) {
		c.sleep(600 * time.Millisecond)
		forte = 0
	}

	if forte < 3 && changliFlickResonance(c, 200*time.Millisecond, false) {
		c.state.changliEnhancedNormal = true
		return
	}
	c.echoWait(1 * time.Second)
}

// changliHeavyRelease mirrors ok-ww Changli.heavy_click_forte():
// flying→heavy → holdHeavyUntil !mouse_forte_full.
func changliHeavyRelease(c combatActor) bool {
	if c.flying() {
		c.heavy(600 * time.Millisecond)
	}
	c.holdHeavyUntil(1200*time.Millisecond, 50*time.Millisecond, func() bool {
		return !c.mouseForteFull()
	})
	c.state.lastHeavy = time.Now()
	return true
}

// changliLiberationAndHeavy mirrors ok-ww Changli.liberation_and_heavy():
// casts liberation, holds heavy during animation (after 1.5s→5.5s boundary),
// waits for mouse_forte_full→drop cycle after return.
func changliLiberationAndHeavy(c combatActor, sendClick bool, waitIfReady time.Duration, timeout time.Duration) bool {
	if !c.liberationAvailable() {
		return false
	}
	start := time.Now()
	clicked := false

	for time.Since(start) < waitIfReady && !c.liberationAvailable() {
		if sendClick {
			c.attack()
		}
		c.sleep(100 * time.Millisecond)
	}

	for c.liberationAvailable() && c.isCurrentChar() {
		if time.Since(start) > timeout {
			return false
		}
		c.forceLiberation()
		clicked = true
		c.sleep(100 * time.Millisecond)
	}
	if !clicked {
		return false
	}

	leaveDeadline := time.Now().Add(400 * time.Millisecond)
	for time.Now().Before(leaveDeadline) && c.isCurrentChar() {
		c.sleep(50 * time.Millisecond)
	}
	if c.isCurrentChar() {
		return false
	}

	freezeStart := time.Now()
	backDeadline := freezeStart.Add(7 * time.Second)
	holding := false
	for time.Now().Before(backDeadline) && !c.isCurrentChar() {
		if sendClick {
			c.attack()
		}
		if !holding && time.Until(backDeadline) < 5500*time.Millisecond {
			c.ctx.GetTasker().GetController().PostTouchDown(0, 640, 360, 1).Wait()
			holding = true
		}
		c.sleep(100 * time.Millisecond)
	}
	if holding {
		c.ctx.GetTasker().GetController().PostTouchUp(0).Wait()
	}
	if !c.isCurrentChar() {
		return false
	}
	c.addFreezeDuration(time.Since(freezeStart))
	c.state.lastLiberation = time.Now()
	c.waitDown(600 * time.Millisecond)
	waitMouseDeadline := time.Now().Add(600 * time.Millisecond)
	for time.Now().Before(waitMouseDeadline) && !c.mouseForteFull() {
		c.sleep(50 * time.Millisecond)
	}
	waitDropDeadline := time.Now().Add(600 * time.Millisecond)
	for time.Now().Before(waitDropDeadline) && c.mouseForteFull() {
		c.sleep(50 * time.Millisecond)
	}
	c.sleep(50 * time.Millisecond)
	return true
}

// changliFlickResonance mirrors ok-ww Changli.flick_resonance():
// if sendClick, attacks until resonance>0; then spams resonance until exhausted or timeout.
func changliFlickResonance(c combatActor, timeout time.Duration, sendClick bool) bool {
	if !c.resonanceAvailable() || c.currentResonance() <= 0.05 {
		return false
	}
	start := time.Now()
	if sendClick {
		for c.currentResonance() <= 0.001 && time.Since(start) < 200*time.Millisecond {
			c.attack()
			c.sleep(100 * time.Millisecond)
		}
	}
	if c.currentResonance() <= 0.001 {
		return false
	}
	clicked := false
	for c.resonanceAvailable() && c.currentResonance() > 0.05 && time.Since(start) < timeout {
		if c.currentResonance() > 0 && c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

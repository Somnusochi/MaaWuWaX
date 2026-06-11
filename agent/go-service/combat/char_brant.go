package combat

import "time"

// performBrant mirrors ok-ww Brant.do_perform():
//
//	intro(1.3s attack, lupa outro→perform_in_outro) → f_break →
//	forte_full→resonance_forte_full(switch) → liberation(0.8s attack) →
//	still_in_liberation→jump_click_loop(1.3s) → forte_full→resonance(switch) → echo → switch
func performBrant(c combatActor) {
	intro := c.recentlyIntroSwitchedIn(1600 * time.Millisecond)
	inOutro := intro && c.switchedFromName("lupa", 1600*time.Millisecond)
	if intro {
		c.attackFor(1300 * time.Millisecond)
	}

	if inOutro && brantPerformInOutro(c) {
		c.requestSwitch()
		return
	}

	c.fBreak()
	if c.forteFull() && c.resonanceAvailable() {
		brantResonanceForteFull(c)
		c.state.lastLiberation = time.Time{}
		c.state.brantLiberationFreeze = 0
		c.state.lastAnchor = time.Now()
		c.state.brantAnchorFreeze = screenAnalyzer.FreezeDuration
		c.requestSwitch()
		return
	}

	if !c.needFastPerform() && !c.forteFull() && brantClickLiberation(c) {
		c.attackFor(800 * time.Millisecond)
	}

	if !brantStillInLiberation(c) {
		if c.echoWait(1 * time.Second) {
			c.requestSwitch()
			return
		}
	}

	brantJumpWithClick(c, 1300*time.Millisecond)
	if c.forteFull() && c.resonanceAvailable() {
		brantResonanceForteFull(c)
		c.state.lastLiberation = time.Time{}
		c.state.brantLiberationFreeze = 0
		c.state.lastAnchor = time.Now()
		c.state.brantAnchorFreeze = screenAnalyzer.FreezeDuration
		c.requestSwitch()
		return
	}
	if !brantStillInLiberation(c) {
		c.echoWait(1 * time.Second)
	}
	c.requestSwitch()
}

// brantStillInLiberation mirrors ok-ww Brant.still_in_liberation():
// true when <12s since last liberation cast.
func brantStillInLiberation(c combatActor) bool {
	return !c.state.lastLiberation.IsZero() &&
		c.freezeElapsed(c.state.lastLiberation, c.state.brantLiberationFreeze) < 12*time.Second
}

// brantClickLiberation mirrors ok-ww Brant.click_liberation():
// casts liberation, records freeze timestamp on success.
func brantClickLiberation(c combatActor) bool {
	if !c.liberationAvailable() {
		return false
	}
	start := time.Now()
	clicked := false
	for time.Since(start) < 800*time.Millisecond && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) {
		c.forceLiberation()
		clicked = true
		c.sleep(100 * time.Millisecond)
	}
	if !finishLiberationCast(c, clicked, 3*time.Second) {
		return false
	}
	c.state.brantLiberationFreeze = screenAnalyzer.FreezeDuration
	return true
}

// brantResonanceForteFull mirrors ok-ww Brant.resonance_forte_full():
// spams resonance key while forte_full and resonance available (up to 1s).
func brantResonanceForteFull(c combatActor) {
	start := time.Now()
	for c.resonanceAvailable() && c.currentResonance() > 0.05 && c.forteFull() && time.Since(start) < 1*time.Second {
		if c.currentResonance() > 0 {
			c.forceSkill()
		}
		c.sleep(100 * time.Millisecond)
	}
}

// brantFlickResonance mirrors ok-ww Brant.flick_resonance():
// attacks until resonance>0 (200ms), then spams resonance key until exhausted (200ms).
func brantFlickResonance(c combatActor) bool {
	if !c.resonanceAvailable() || c.currentResonance() <= 0.05 {
		return false
	}
	// Step 1: attack until resonance > 0 (timeout 200ms)
	start := time.Now()
	for c.currentResonance() <= 0.001 && time.Since(start) < 200*time.Millisecond {
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	// Step 2: spam resonance key until exhausted (timeout 200ms)
	if c.currentResonance() > 0.001 {
		start = time.Now()
		for c.resonanceAvailable() && c.currentResonance() > 0.05 && time.Since(start) < 200*time.Millisecond {
			if c.currentResonance() > 0 {
				c.forceSkill()
			}
			c.sleep(100 * time.Millisecond)
		}
		return true
	}
	return false
}

// brantJumpWithClick mirrors ok-ww Brant.click_jump_with_click():
// alternates jump and attack for duration, starting with flick_resonance if grounded.
func brantJumpWithClick(c combatActor, delay time.Duration) {
	click := false
	if !c.flying() {
		brantFlickResonance(c)
		c.sleep(200 * time.Millisecond)
		click = true
	}
	start := time.Now()
	for time.Since(start) < delay {
		if click {
			c.attack()
		} else {
			c.jump()
		}
		click = !click
		c.sleep(100 * time.Millisecond)
	}
}

// brantPerformInOutro mirrors ok-ww Brant.perform_in_outro():
// during lupa outro: loop forte→resonance / liberation / jump+attack → echo.
func brantPerformInOutro(c combatActor) bool {
	start := time.Now()
	timeout := 1500 * time.Millisecond
	if brantStillInLiberation(c) {
		timeout = 10 * time.Second
	}
	for time.Since(start) < timeout {
		if c.forteFull() && c.resonanceAvailable() {
			brantResonanceForteFull(c)
			c.state.lastAnchor = time.Now()
			c.state.brantAnchorFreeze = screenAnalyzer.FreezeDuration
			c.state.lastLiberation = time.Time{}
			c.state.brantLiberationFreeze = 0
			c.waitDown(4 * time.Second)
			if timeout == 10*time.Second {
				break
			}
		}
		if brantClickLiberation(c) {
			start = time.Now()
			timeout = 10 * time.Second
		}
		if !c.flying() {
			brantFlickResonance(c)
			c.sleep(200 * time.Millisecond)
		}
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	c.echo()
	return true
}

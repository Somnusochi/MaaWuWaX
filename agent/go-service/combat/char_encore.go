package combat

import "time"

// performEncore mirrors ok-ww Encore.do_perform():
//
//	intro: if 6s<elapsed_liberation<10s+mouse_forte_full→heavy(1.2s)→switch / else wait_down →
//	still_in_liberation→n4→switch /
//	resonance(!can_resonance_step2)→switch /
//	!need_fast_perform+liberation→n4→switch / echo→switch
func performEncore(c combatActor) {
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		elapsed := c.freezeElapsed(c.state.lastLiberation, c.state.encoreLiberationFreeze)
		if elapsed > 6*time.Second && elapsed < 10*time.Second && c.mouseForteFull() {
			c.heavy(1200 * time.Millisecond)
			c.sleep(100 * time.Millisecond)
			c.state.lastHeavy = time.Now()
			c.state.encoreHeavyFreeze = screenAnalyzer.FreezeDuration
			c.requestSwitch()
			return
		}
		c.waitDown(1200 * time.Millisecond)
	}

	if c.freezeElapsed(c.state.lastLiberation, c.state.encoreLiberationFreeze) < 9500*time.Millisecond {
		encorePerformN4(c)
		c.requestSwitch()
		return
	}

	canResonanceStep2 := encoreCanResonanceStep2(c, 4*time.Second)
	if encoreClickResonance(c) && !canResonanceStep2 {
		c.state.lastResonance = time.Now()
		c.state.encoreResonanceFreeze = screenAnalyzer.FreezeDuration
		c.requestSwitch()
		return
	}

	// KNOWN_DIFF: Python uses full click_liberation with animation detection;
	// Go uses simplified retry loop (framework limitation).
	if !c.needFastPerform() && !c.isOpenWorldAutoCombat() && encoreTryLiberation(c, 400*time.Millisecond) {
		encorePerformN4(c)
		c.requestSwitch()
		return
	}

	if c.echoWait(1 * time.Second) {
		c.requestSwitch()
		return
	}

	c.requestSwitch()
}

// encoreCanResonanceStep2 mirrors ok-ww Encore.can_resonance_step2():
// true when <delay has passed since last_resonance.
func encoreCanResonanceStep2(c combatActor, delay time.Duration) bool {
	return c.freezeElapsed(c.state.lastResonance, c.state.encoreResonanceFreeze) < delay
}

// encorePerformN4 mirrors ok-ww Encore.n4():
// resonance→attack(2.7s) / resonance_failed→attack(2.4s) /
//   liberation<6s: attack n4 / mouse_forte_full: heavy / else: resonance.
func encorePerformN4(c combatActor) {
	duration := 2400 * time.Millisecond
	if encoreClickResonance(c) {
		duration = 2700 * time.Millisecond
	}
	if c.freezeElapsed(c.state.lastLiberation, c.state.encoreLiberationFreeze) < 6*time.Second {
		c.attackFor(duration)
		return
	}
	if c.mouseForteFull() {
		c.heavy(600 * time.Millisecond)
		c.state.lastHeavy = time.Now()
		c.state.encoreHeavyFreeze = screenAnalyzer.FreezeDuration
		return
	}
	encoreClickResonance(c)
}

// encoreClickResonance mirrors ok-ww Encore.click_resonance():
// casts resonance while available for up to 15s.
// IMPORTANT: Python's click_resonance() records the cast time in last_res,
// NOT last_resonance. Encore.last_resonance is a separate step2-tracking
// field that must NOT be overwritten by routine resonance casts. We save
// and restore it around the cast so can_resonance_step2 windows stay correct.
func encoreClickResonance(c combatActor) bool {
	if c.currentResonance() <= 0.05 {
		return false
	}
	savedResonance := c.state.lastResonance
	savedResonanceFreeze := c.state.encoreResonanceFreeze

	start := time.Now()
	clicked := false
	for c.currentResonance() > 0.05 && time.Since(start) < 15*time.Second {
		if c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}

	c.state.lastResonance = savedResonance
	c.state.encoreResonanceFreeze = savedResonanceFreeze
	return clicked
}

// encoreTryLiberation mirrors ok-ww Encore.click_liberation(wait_if_cd_ready=0.4):
// attempts liberation with deadline retry window, recording timestamp on success.
func encoreTryLiberation(c combatActor, wait time.Duration) bool {
	if wait <= 0 {
		if c.liberation() {
			c.state.lastLiberation = time.Now()
			c.state.encoreLiberationFreeze = screenAnalyzer.FreezeDuration
			return true
		}
		return false
	}
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		if c.liberation() {
			c.state.lastLiberation = time.Now()
			c.state.encoreLiberationFreeze = screenAnalyzer.FreezeDuration
			return true
		}
		c.sleep(100 * time.Millisecond)
	}
	return false
}

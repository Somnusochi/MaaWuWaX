package combat

import "time"

// performIuno mirrors ok-ww Iuno.do_perform():
//
//	wait_down → do_everything: echo → loop(timeout=1.5s+4s intro):
//	  heavy(iuno_heavy icon)→C6 extends / jump(iuno_jump icon)→timeout+3s /
//	  liberation(>20s cd)→reset / alternating resonance+attack → switch
func performIuno(c combatActor) {
	c.waitDown(1200 * time.Millisecond)
	start := time.Now()
	startFreeze := screenAnalyzer.FreezeDuration
	timeout := 1500 * time.Millisecond
	intro := c.recentlyIntroSwitchedIn(1600 * time.Millisecond)
	if intro {
		timeout += 4 * time.Second
	}
	c.echoWait(1 * time.Second)
	lastActionResonance := false
	c6Performed := false
	jumped := false
	for c.freezeElapsed(start, startFreeze) < timeout {
		cycleStart := time.Now()
		cycleFreeze := screenAnalyzer.FreezeDuration
		heavySuccess := false
		for c.freezeElapsed(c.state.lastHeavy, c.state.iunoHeavyFreeze) > 20*time.Second && c.iunoHeavyReady() {
			c.sleep(50 * time.Millisecond)
			c.heavy(600 * time.Millisecond)
			c.sleep(50 * time.Millisecond)
			heavySuccess = true
		}
		if heavySuccess {
			c.state.lastHeavy = time.Now()
			c.state.iunoHeavyFreeze = screenAnalyzer.FreezeDuration
			if c.iunoC6() && !c6Performed {
				c6Performed = true
				start = time.Now()
				startFreeze = screenAnalyzer.FreezeDuration
				timeout = 5 * time.Second
				continue
			}
			c.requestSwitch()
			return
		}
		if !jumped && c.iunoJumpReady() {
			for c.iunoJumpReady() {
				c.jump()
				c.sleep(100 * time.Millisecond)
			}
			timeout += 3 * time.Second
			jumped = true
			if !intro {
				c.requestSwitch()
				return
			}
			continue
		}
		if c.freezeElapsed(c.state.lastLiberation, c.state.iunoLiberationFreeze) > 20*time.Second && iunoClickLiberation(c) {
			start = time.Now()
			startFreeze = screenAnalyzer.FreezeDuration
			timeout = 3 * time.Second
		}
		if lastActionResonance {
			c.attack()
		} else {
			c.forceSkill()
		}
		lastActionResonance = !lastActionResonance
		if wait := 100*time.Millisecond - c.freezeElapsed(cycleStart, cycleFreeze); wait > 0 {
			c.sleep(wait)
		}
	}
	c.requestSwitch()
}

// iunoClickLiberation mirrors ok-ww Iuno.click_liberation(wait_if_cd_ready=0):
// standard liberation cast with finishLiberationCast, records freeze timestamp.
func iunoClickLiberation(c combatActor) bool {
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
	c.state.iunoLiberationFreeze = screenAnalyzer.FreezeDuration
	return true
}

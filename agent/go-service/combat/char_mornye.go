package combat

import "time"

// performMornye mirrors ok-ww Mornye.do_perform():
//
//	intro(1.33s attack) → !on_air+combo_limit(heavy<23s)→echo/resonance/attack → switch /
//	!on_air→ground_actions(click+resonance loop, heavy with right_click) /
//	on_air→air_actions(elbow_strike_right_click / liberation / heavy_click_forte / resonance → switch)
func performMornye(c combatActor) {
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		c.attackFor(1330 * time.Millisecond)
	}

	if !mornyeOnAir(c) && !c.state.lastHeavy.IsZero() && c.freezeElapsed(c.state.lastHeavy, c.state.mornyeHeavyFreeze) < 23*time.Second {
		if c.echoWait(1 * time.Second) {
			c.requestSwitch()
			return
		}
		if mornyeClickResonance(c) {
			c.requestSwitch()
			return
		}
		c.attackFor(100 * time.Millisecond)
		c.requestSwitch()
		return
	}

	if !mornyeOnAir(c) {
		mornyeGroundActions(c)
	}
	if mornyeOnAir(c) && mornyeAirActions(c) {
		return
	}
	c.requestSwitch()
}

func mornyeOnAir(c combatActor) bool {
	return c.hasLongAction2()
}

// mornyeGroundActions mirrors ok-ww Mornye.not_on_air_actions():
// click+resonance loop(10s), heavy_attack when mouse_forte_full, right_click dodge first time.
func mornyeGroundActions(c combatActor) {
	start := time.Now()
	tryDodge := true
	for time.Since(start) < 10*time.Second && !mornyeOnAir(c) {
		c.attack()
		mornyeClickResonance(c)
		c.sleep(100 * time.Millisecond)
		if c.mouseForteFull() {
			if tryDodge {
				c.rightClick()
				tryDodge = false
			}
			c.heavy(600 * time.Millisecond)
			c.sleep(300 * time.Millisecond)
		}
	}
}

// mornyeAirActions mirrors ok-ww Mornye.on_air_actions():
// 10s loop: elbow_strike→right_click / liberation / mouse_forte_full→heavy_click_forte→echo→switch / resonance+attack.
func mornyeAirActions(c combatActor) bool {
	detectReady := c.currentEcho() > 0.05
	start := time.Now()
	for time.Since(start) < 10*time.Second && mornyeOnAir(c) {
		if mornyeDetectElbowStrike(c, detectReady) {
			waitUntil := time.Now().Add(1500 * time.Millisecond)
			for time.Now().Before(waitUntil) && mornyeDetectElbowStrike(c, detectReady) {
				c.rightClickFor(50 * time.Millisecond)
			}
		}
		mornyeClickLiberation(c)
		if mornyeOnAir(c) && c.mouseForteFull() {
			c.holdHeavyUntil(2*time.Second, 100*time.Millisecond, func() bool {
				return !c.mouseForteFull() || mornyeDetectElbowStrike(c, detectReady)
			})
			if mornyeDetectElbowStrike(c, detectReady) {
				continue
			}
			if !mornyeWaitConcertoFull(c, 1500*time.Millisecond) {
				c.echoWait(200 * time.Millisecond)
			}
			c.state.lastHeavy = time.Now()
			c.state.mornyeHeavyFreeze = screenAnalyzer.FreezeDuration
			c.requestSwitch()
			return true
		}
		mornyeClickResonance(c)
		c.attack()
		c.sleep(10 * time.Millisecond)
	}
	return false
}

func mornyeClickLiberation(c combatActor) bool {
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

func mornyeClickResonance(c combatActor) bool {
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

func mornyeWaitConcertoFull(c combatActor, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if screenAnalyzer.ConcertoPct >= 1.0 {
			return true
		}
		c.sleep(100 * time.Millisecond)
	}
	return screenAnalyzer.ConcertoPct >= 1.0
}

func mornyeDetectElbowStrike(c combatActor, ready bool) bool {
	return ready && c.currentEcho() <= 0.05
}

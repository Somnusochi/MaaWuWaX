package combat

import (
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
)

// performCarlotta mirrors ok-ww Carlotta.do_perform():
//
//	team_has_zhezhi → performCarlottaInterlock / normal:
//	  intro(1.3s attack) → heavy_click_forte(switch) →
//	  liberation loop(auto_dodge, click_liberation_1) → echo → resonance(heavy first if bullet=0) → echo → switch
func performCarlotta(c combatActor) {
	if c.teamHasAny("zhezhi", "zhezhi2") {
		performCarlottaInterlock(c)
		return
	}

	bullet := 0
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		bullet = 1
		c.attackFor(1300 * time.Millisecond)
	}
	if carlottaHeavyClickForte(c) {
		c.state.liberationReady = false
		c.requestSwitch()
		return
	}
	if (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) && !c.needFastPerform() {
		// Python loops while self.liberation_available() to fire multiple shots
		for screenAnalyzer.Liberation || c.currentLiberation() > 0.05 {
			carlottaCastLiberation(c, carlottaShouldPressW(c))
		}
		carlottaEchoWait(c, time.Second)
		c.requestSwitch()
		return
	}
	if c.currentResonance() > 0.05 {
		if bullet == 0 {
			c.heavy(600 * time.Millisecond)
		}
		if carlottaClickResonance(c) {
			c.requestSwitch()
			return
		}
	}
	if carlottaEchoWait(c, time.Second) {
		c.requestSwitch()
		return
	}
	c.attackFor(310 * time.Millisecond)
	c.requestSwitch()
}

// performCarlottaInterlock mirrors ok-ww Carlotta.do_perform_interlock():
// zhezhi interlock: intro → forte<4+resonance → heavy_click_forte for liberation_ready → liberation → echo → switch.
func performCarlottaInterlock(c combatActor) {
	// carlottaContinueLiberation = continueLiberation
	// liberationReady = heavy-confirmed liberation setup
	bullet := 0
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		bullet = 1
		c.attackFor(1300 * time.Millisecond)
		if c.switchedFromAny(1800*time.Millisecond, "zhezhi", "zhezhi2") {
			performCarlottaOutro(c)
			c.requestSwitch()
			return
		}
	}
	// KNOWN_DIFF: Python get_forte() returns 0-4 discrete FFT tiers; Go uses mouseForteFull()/currentForte() float percentage
	if !c.mouseForteFull() && c.currentResonance() > 0.05 && !c.state.liberationReady {
		if bullet == 0 {
			c.heavy(600 * time.Millisecond)
		}
		if carlottaClickResonance(c) {
			c.requestSwitch()
			return
		}
	}
	if carlottaReady(c) {
		c.state.carlottaContinueLiberation = false
	}
	if carlottaHeavyClickForte(c) {
		c.state.liberationReady = true
		c.requestSwitch()
		return
	}
	if (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) && c.state.carlottaContinueLiberation {
		for screenAnalyzer.Liberation || c.currentLiberation() > 0.05 {
			if carlottaCastLiberation(c, false) {
				c.state.carlottaContinueLiberation = false
				c.state.liberationReady = false
			}
		}
	}
	if carlottaEchoWait(c, time.Second) {
		c.attackFor(310 * time.Millisecond)
		c.requestSwitch()
		return
	}
	c.attackFor(310 * time.Millisecond)
	c.requestSwitch()
}

// performCarlottaOutro mirrors ok-ww Carlotta.do_perform_outro():
// during zhezhi outro: build forte → heavy_click_forte → liberation+resonance loop → echo → set continue_liberation.
func performCarlottaOutro(c combatActor) {
	if !c.state.liberationReady {
		for !c.mouseForteFull() && c.performElapsed() < 6*time.Second {
			if c.currentResonance() > 0.05 {
				carlottaClickResonance(c)
			} else {
				c.attack()
			}
			c.sleep(100 * time.Millisecond)
		}
	}
	if carlottaHeavyClickForte(c) {
		c.state.liberationReady = true
	}

	clickedResonance := false
	castLiberation := false
	if c.state.liberationReady {
		for c.performElapsed() < 14*time.Second {
			if (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) && !castLiberation {
				// Python: while self.liberation_available() to fire multiple shots
				for screenAnalyzer.Liberation || c.currentLiberation() > 0.05 {
					if carlottaCastLiberation(c, false) {
						c.state.liberationReady = false
						castLiberation = true
					}
				}
				if castLiberation {
					c.sleep(200 * time.Millisecond)
				}
			}
			if c.currentResonance() > 0.05 {
				if carlottaClickResonance(c) {
					c.attackFor(800 * time.Millisecond)
					clickedResonance = true
				}
			}
			c.attackFor(100 * time.Millisecond)
			if !castLiberation && !screenAnalyzer.Liberation && c.currentLiberation() <= 0.05 && c.currentResonance() <= 0.05 && clickedResonance {
				break
			}
			c.sleep(100 * time.Millisecond)
		}
	}
	if carlottaEchoWait(c, 2*time.Second) {
		// ok-ww: sets switch_lock but wait_switch() is dead code — no-op here
	}
	c.state.carlottaContinueLiberation = !castLiberation
}

// KNOWN_DIFF: Python get_forte() returns 0-4 discrete FFT tiers; Go uses currentForte() float percentage
// carlottaReady mirrors ok-ww Carlotta.get_ready():
// true when liberation_ready, mouse_forte_full, forte>0.5, or resonance+forte>0.02.
func carlottaReady(c combatActor) bool {
	if c.state.liberationReady || c.mouseForteFull() {
		return true
	}
	forte := c.currentForte()
	if forte > 0.5 {
		return true
	}
	return c.currentResonance() > 0.05 && forte > 0.02
}

// carlottaHeavyClickForte mirrors ok-ww Carlotta.heavy_click_forte():
// holds heavy while mouse_forte_full for up to 1.2s.
func carlottaHeavyClickForte(c combatActor) bool {
	if !c.mouseForteFull() {
		return false
	}
	c.holdHeavyUntil(1200*time.Millisecond, 100*time.Millisecond, func() bool {
		return !c.mouseForteFull()
	})
	success := !c.mouseForteFull()
	c.sleep(50 * time.Millisecond)
	return success
}

// carlottaShouldPressW mirrors ok-ww Carlotta.decide_teammate().press_w:
// true in Farm 4C Echo task; holds W during liberation cast.
func carlottaShouldPressW(c combatActor) bool {
	return c.action != nil && c.action.currentTaskName == "Farm 4C Echo in Dungeon/World"
}

// carlottaCastLiberation mirrors ok-ww Carlotta.click_liberation_1() + click_liberation():
// casts liberation with optional W key hold, auto_dodge during animation, 7s return wait.
func carlottaCastLiberation(c combatActor, holdForward bool) bool {
	autoDodgeStart := time.Time{}
	start := time.Now()
	clicked := false
	ctrl := c.ctx.GetTasker().GetController()
	if holdForward {
		ctrl.PostKeyDown(keycode.MustCode("W")).Wait()
		defer ctrl.PostKeyUp(keycode.MustCode("W")).Wait()
	}
	for (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) && time.Since(start) < 400*time.Millisecond {
		if !autoDodgeStart.IsZero() && time.Since(autoDodgeStart) > 500*time.Millisecond && c.flying() {
			c.waitDown(1200 * time.Millisecond)
		}
		if c.forceLiberation() {
			clicked = true
			if autoDodgeStart.IsZero() {
				autoDodgeStart = time.Now()
			}
		}
		c.sleep(100 * time.Millisecond)
	}
	if !clicked {
		retryDeadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(retryDeadline) && c.currentLiberation() > 0.001 {
			if c.forceLiberation() {
				clicked = true
				if autoDodgeStart.IsZero() {
					autoDodgeStart = time.Now()
				}
			}
			c.sleep(50 * time.Millisecond)
		}
		if !clicked {
			return false
		}
	}
	leaveDeadline := time.Now().Add(400 * time.Millisecond)
	for time.Now().Before(leaveDeadline) {
		if !c.isCurrentChar() {
			break
		}
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	if c.isCurrentChar() {
		return false
	}
	freezeStart := time.Now()
	backDeadline := time.Now().Add(7 * time.Second)
	for time.Now().Before(backDeadline) && !c.isCurrentChar() {
		c.sleep(50 * time.Millisecond)
	}
	if !c.isCurrentChar() {
		return false
	}
	c.addFreezeDuration(time.Since(freezeStart))
	c.state.lastLiberation = time.Now()
	return true
}

// carlottaClickResonance mirrors ok-ww Carlotta.click_resonance():
// alternates attack/resonance clicks for up to 10s.
func carlottaClickResonance(c combatActor) bool {
	start := time.Now()
	clicked := false
	clickAttack := false
	for c.currentResonance() > 0.05 && time.Since(start) < 10*time.Second {
		if clickAttack {
			c.attack()
			clickAttack = false
		} else if c.forceSkill() {
			clicked = true
			clickAttack = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

// carlottaEcho mirrors ok-ww Carlotta.click_echo():
// sends echo via rotation task if available.
func carlottaEcho(c combatActor) bool {
	if c.currentEcho() <= 0.05 {
		return false
	}
	c.run("Combat_RotationEcho")
	c.state.lastEcho = time.Now()
	return true
}

// carlottaEchoWait calls carlottaEcho with a deadline-based retry window.
func carlottaEchoWait(c combatActor, wait time.Duration) bool {
	if wait <= 0 {
		return carlottaEcho(c)
	}
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		if carlottaEcho(c) {
			return true
		}
		c.sleep(100 * time.Millisecond)
	}
	return carlottaEcho(c)
}

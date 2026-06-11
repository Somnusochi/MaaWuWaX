package combat

import (
	"strings"
	"time"
)

// performPhrolova mirrors ok-ww Phrolova.do_perform():
//
//	reset last_liberation → intro(1.7s attack+right_click 0.1s, cantarella outro flag) →
//	flying→wait_down → liberation(early→switch) → heavy_and_liber →
//	resonance→attack→wait_resonance_end→echo → loop(4-16s):
//	  liberation / flying→dodge / heavy_and_liber / resonance → switch
func performPhrolova(c combatActor) {
	// Match ok-ww: reset lastLiberation at perform start (≡ self.last_liberation = -1).
	// freezeElapsed returns MaxInt64 for zero-value, so 24s lock is disabled until
	// liberation is successfully cast WITHIN this perform — same as ok-ww.
	c.state.lastLiberation = time.Time{}
	c.state.lastLiberationFreeze = 0
	performUnderOutro := false
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		c.state.phrolovaResReady = false
		c.attackFor(1700 * time.Millisecond)
		c.rightClickFor(100 * time.Millisecond)
		if c.switchedFromName("cantarella", 1800*time.Millisecond) {
			performUnderOutro = true
		}
	}
	if c.flying() {
		c.waitDown(1500 * time.Millisecond)
	}
	if phrolovaClickLiberation(c) {
		phrolovaNightmareLiberationFollowUp(c, 1*time.Second)
		c.requestSwitch()
		return
	}
	if phrolovaHeavyAndLiber(c) {
		c.requestSwitch()
		return
	}
	if c.state.phrolovaResReady || phrolovaResAvailable(c) {
		c.attackFor(100 * time.Millisecond)
		phrolovaClickResonance(c)
		c.attackFor(100 * time.Millisecond)
		phrolovaWaitResonanceEnd(c, 300*time.Millisecond)
		if !c.echoWait(1 * time.Second) {
			c.rightClickFor(100 * time.Millisecond)
		}
	}
	c.state.phrolovaResReady = false
	start := time.Now()
	for {
		if performUnderOutro {
			if c.performElapsed() >= 16*time.Second {
				break
			}
		} else if time.Since(start) >= 4*time.Second {
			break
		}
		if phrolovaClickLiberation(c) {
			phrolovaNightmareLiberationFollowUp(c, 1500*time.Millisecond)
			c.requestSwitch()
			return
		}
		if c.flying() {
			shorekeeperAutoDodge(c, func() bool { return c.flying() })
		}
		if phrolovaHeavyAndLiber(c) {
			c.requestSwitch()
			return
		}
		if time.Since(start) > 1*time.Second && phrolovaResAvailableInMode(c, performUnderOutro) {
			if performUnderOutro {
				c.attackFor(300 * time.Millisecond)
				if phrolovaClickResonanceInMode(c, performUnderOutro) {
					c.attackFor(100 * time.Millisecond)
					phrolovaWaitResonanceEnd(c, 300*time.Millisecond)
				}
				if !c.echoWait(1 * time.Second) {
					c.rightClickFor(100 * time.Millisecond)
				}
			} else {
				c.state.phrolovaResReady = true
				break
			}
		}
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	c.requestSwitch()
}

func phrolovaClickResonance(c combatActor) bool {
	return phrolovaClickResonanceInMode(c, false)
}

func phrolovaClickResonanceInMode(c combatActor, performUnderOutro bool) bool {
	start := time.Now()
	clicked := false
	for phrolovaResAvailableInMode(c, performUnderOutro) && time.Since(start) < 15*time.Second {
		if c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

func phrolovaClickLiberation(c combatActor) bool {
	if !c.param.UseLiberation {
		return false
	}
	start := time.Now()
	clicked := false
	for time.Since(start) < 800*time.Millisecond && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) {
		c.forceLiberation()
		clicked = true
		c.sleep(100 * time.Millisecond)
	}
	if !clicked {
		retryDeadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(retryDeadline) && c.currentLiberation() > 0.001 {
			c.forceLiberation()
			clicked = true
			c.sleep(100 * time.Millisecond)
		}
	}
	return finishLiberationCast(c, clicked, 7*time.Second)
}

func phrolovaWaitResonanceEnd(c combatActor, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) && phrolovaResAvailable(c) {
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
}

func phrolovaResAvailable(c combatActor) bool {
	return phrolovaResAvailableInMode(c, false)
}

func phrolovaResAvailableInMode(c combatActor, performUnderOutro bool) bool {
	if !c.resonanceNoCD() {
		return false
	}
	if c.freezeElapsed(c.state.lastResonance, c.state.lastResonanceFreeze) < 2*time.Second {
		return false
	}
	if performUnderOutro {
		return !c.flying()
	}
	return true
}

// phrolovaHeavyAndLiber mirrors ok-ww Phrolova.heavy_and_liber():
// heavy_click_forte → wait 3s for liberation.
func phrolovaHeavyAndLiber(c combatActor) bool {
	if !c.mouseForteFull() {
		return false
	}
	c.holdHeavyUntil(600*time.Millisecond, 100*time.Millisecond, func() bool {
		return !c.mouseForteFull()
	})
	waitDeadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(waitDeadline) {
		if phrolovaClickLiberation(c) {
			return true
		}
		c.sleep(100 * time.Millisecond)
	}
	return false
}

func phrolovaNightmareLiberationFollowUp(c combatActor, duration time.Duration) {
	if duration <= 0 || c.action == nil {
		return
	}
	taskName := strings.ToLower(c.action.currentTaskName)
	if !strings.Contains(taskName, "nightmare") && !strings.Contains(taskName, "梦魇") {
		return
	}
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		c.forceLiberation()
		c.sleep(100 * time.Millisecond)
	}
}

package combat

import "time"

// performDefault mirrors ok-ww BaseChar.do_perform():
//
//	wait_intro(1.2s) → echo → liberation → resonance → [if !res] heavy_click_forte → switch
func performDefault(c combatActor) {
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		c.waitIntro(1200*time.Millisecond, true)
	}
	c.echoImmediate()
	defaultClickLiberation(c)
	if !defaultClickResonance(c) {
		defaultHeavyClickForte(c)
	}
	c.requestSwitch()
}

func defaultClickLiberation(c combatActor) bool {
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
	if !clicked {
		retryDeadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(retryDeadline) && c.currentLiberation() > 0.001 {
			if c.forceLiberation() {
				clicked = true
			}
			c.sleep(50 * time.Millisecond)
		}
	}
	return finishLiberationCast(c, clicked, 7*time.Second)
}

func defaultClickResonance(c combatActor) bool {
	if !c.resonanceAvailable() {
		return false
	}
	start := time.Now()
	clicked := false
	lastOp := "click"
	for c.resonanceChainAvailable() && time.Since(start) < 15*time.Second {
		if lastOp == "resonance" {
			c.attack()
			lastOp = "click"
		} else if c.currentResonance() > 0 && c.forceSkill() {
			clicked = true
			lastOp = "resonance"
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

func defaultHeavyClickForte(c combatActor) bool {
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

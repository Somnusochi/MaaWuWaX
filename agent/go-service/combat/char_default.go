package combat

import "time"

// performDefault mirrors ok-ww BaseChar.do_perform():
//
//	wait_intro(1.2s) → echo → liberation → resonance → [if !res] heavy_click_forte → switch
func performDefault(c combatActor) {
	if c.recentlySwitchedIn(1800 * time.Millisecond) {
		c.attackFor(1200 * time.Millisecond)
	}
	c.echo()
	c.liberation()
	if !c.skill() {
		c.heavy(600 * time.Millisecond)
	}
	c.requestSwitch()
}

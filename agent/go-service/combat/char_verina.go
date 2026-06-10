package combat

import "time"

func performVerina(c combatActor) {
	if c.recentlySwitchedIn(1800 * time.Millisecond) {
		c.attackFor(1200 * time.Millisecond)
	}
	if c.liberation() {
		c.attackFor(300 * time.Millisecond)
		c.requestSwitch()
		return
	}
	if c.skill() {
		c.attackFor(200 * time.Millisecond)
		c.requestSwitch()
		return
	}
	c.echo()
	c.requestSwitch()
}

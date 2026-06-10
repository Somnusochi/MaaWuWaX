package combat

import "time"

func performShorekeeper(c combatActor) {
	if c.recentlySwitchedIn(1800 * time.Millisecond) {
		c.attackFor(1200 * time.Millisecond)
	} else {
		c.attackFor(300 * time.Millisecond)
	}
	c.echo()
	c.liberation()
	if !c.skill() {
		c.heavy(500 * time.Millisecond)
	}
	c.requestSwitch()
}

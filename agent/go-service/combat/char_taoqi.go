package combat

import "time"

func performTaoqi(c combatActor) {
	if c.recentlySwitchedIn(3 * time.Second) {
		c.attackFor(2500 * time.Millisecond)
	}
	c.liberation()
	c.skill()
	c.echo()
	c.requestSwitch()
}

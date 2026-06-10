package combat

import "time"

func performBaizhi(c combatActor) {
	if c.recentlySwitchedIn(1800 * time.Millisecond) {
		c.attackFor(1200 * time.Millisecond)
	}
	c.liberation()
	c.skill()
	c.echo()
	c.requestSwitch()
}

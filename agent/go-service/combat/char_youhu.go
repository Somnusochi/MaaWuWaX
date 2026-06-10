package combat

import "time"

func performYouhu(c combatActor) {
	if c.recentlySwitchedIn(1200 * time.Millisecond) {
		c.attackFor(500 * time.Millisecond)
	}
	c.echo()
	if c.liberation() {
		c.attackFor(200 * time.Millisecond)
		c.requestSwitch()
		return
	}
	c.skill()
	c.requestSwitch()
}

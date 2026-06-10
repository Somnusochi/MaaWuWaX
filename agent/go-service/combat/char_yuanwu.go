package combat

import "time"

func performYuanwu(c combatActor) {
	if c.liberation() {
		c.skill()
		c.requestSwitch()
		return
	}
	if c.recentlySwitchedIn(1800 * time.Millisecond) {
		c.attackFor(1200 * time.Millisecond)
	}
	c.skill()
	c.echo()
	c.requestSwitch()
}

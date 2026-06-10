package combat

import "time"

func performJianxin(c combatActor) {
	if c.recentlySwitchedIn(1500 * time.Millisecond) {
		c.attackFor(1 * time.Second)
	} else {
		c.attackFor(400 * time.Millisecond)
	}
	c.liberation()
	if !c.skill() {
		c.heavy(450 * time.Millisecond)
	}
	c.echo()
	c.requestSwitch()
}

package combat

import "time"

func performDenia(c combatActor) {
	if !c.state.flag {
		c.attackFor(1200 * time.Millisecond)
		c.state.flag = true
	} else {
		c.attackFor(450 * time.Millisecond)
	}
	if c.skill() {
		c.attackFor(300 * time.Millisecond)
	}
	if c.liberation() {
		c.skill()
		c.attackFor(600 * time.Millisecond)
	}
	c.echo()
	c.requestSwitch()
}

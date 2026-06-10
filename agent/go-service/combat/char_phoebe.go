package combat

import "time"

func performPhoebe(c combatActor) {
	if !c.state.flag {
		c.attackFor(1500 * time.Millisecond)
		c.state.flag = true
	} else {
		c.sleep(10 * time.Millisecond)
	}

	c.echo()
	if c.liberation() {
		c.state.flag2 = !c.state.flag2
		c.attackFor(300 * time.Millisecond)
	}
	c.heavy(500 * time.Millisecond)
	if c.skill() {
		if c.state.flag2 {
			c.attackFor(1200 * time.Millisecond)
		} else {
			c.attackFor(700 * time.Millisecond)
		}
	}
	c.echo()
	c.requestSwitch()
}

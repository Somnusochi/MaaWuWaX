package combat

import "time"

func performHavocRover(c combatActor) {
	if !c.state.flag {
		c.attackFor(900 * time.Millisecond)
		c.state.flag = true
	} else {
		c.sleep(10 * time.Millisecond)
	}

	c.heavy(600 * time.Millisecond)
	c.echo()
	if c.liberation() {
		c.skill()
		c.attackFor(800 * time.Millisecond)
		c.requestSwitch()
		return
	}
	if c.skill() {
		c.attackFor(1400 * time.Millisecond)
	} else {
		c.attackFor(1 * time.Second)
	}
	c.requestSwitch()
}

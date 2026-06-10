package combat

import "time"

func performCiaccona(c combatActor) {
	if !c.state.flag {
		c.attackFor(1500 * time.Millisecond)
		c.state.flag = true
	} else {
		c.attackFor(400 * time.Millisecond)
	}
	c.echo()
	waitAfterSkill := c.skill()
	if c.state.flag2 {
		c.heavy(700 * time.Millisecond)
		waitAfterSkill = true
	}
	if waitAfterSkill {
		c.sleep(400 * time.Millisecond)
	}
	if c.liberation() {
		c.state.flag2 = true
		c.attackFor(600 * time.Millisecond)
	}
	c.requestSwitch()
}

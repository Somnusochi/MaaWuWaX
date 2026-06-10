package combat

import "time"

func performBrant(c combatActor) {
	if !c.state.flag {
		c.attackFor(1300 * time.Millisecond)
		c.state.flag = true
	}
	if c.skill() {
		c.state.lastLiberation = time.Time{}
		c.requestSwitch()
		return
	}
	if c.liberation() {
		c.attackFor(800 * time.Millisecond)
	}
	if time.Since(c.state.lastLiberation) < 12*time.Second {
		c.jumpAttackFor(1300 * time.Millisecond)
		if c.skill() {
			c.requestSwitch()
			return
		}
	}
	c.echo()
	c.requestSwitch()
}

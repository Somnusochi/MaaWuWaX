package combat

import "time"

func performChisa(c combatActor) {
	if c.slot.Role == roleHealer {
		c.echo()
		if c.liberation() {
			c.state.lastBuff = time.Now()
			c.requestSwitch()
			return
		}
		c.skill()
		c.requestSwitch()
		return
	}
	if !c.state.flag {
		c.attackFor(800 * time.Millisecond)
		c.state.flag = true
	}
	c.echo()
	if c.liberation() {
		c.attackFor(200 * time.Millisecond)
	}
	if c.skill() {
		c.attackFor(300 * time.Millisecond)
	}
	c.heavy(1200 * time.Millisecond)
	c.requestSwitch()
}

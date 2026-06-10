package combat

import "time"

func performChixia(c combatActor) {
	if !c.state.flag {
		c.attackFor(900 * time.Millisecond)
		c.state.flag = true
	}
	if c.skill() {
		c.attackFor(700 * time.Millisecond)
	}
	if c.liberation() {
		c.attackFor(900 * time.Millisecond)
	}
	c.echo()
	c.requestSwitch()
}

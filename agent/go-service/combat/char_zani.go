package combat

import "time"

func performZani(c combatActor) {
	if time.Since(c.state.lastPerform) > 1500*time.Millisecond {
		c.attackFor(1300 * time.Millisecond)
	}
	if c.liberation() {
		c.state.flag = true
		c.attackFor(650 * time.Millisecond)
		c.heavy(600 * time.Millisecond)
		c.requestSwitch()
		return
	}
	c.echo()
	if c.skill() {
		c.attackFor(450 * time.Millisecond)
	}
	if c.state.flag {
		c.heavy(500 * time.Millisecond)
		c.state.flag = false
	}
	c.requestSwitch()
}

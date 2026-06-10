package combat

import "time"

func performCarlotta(c combatActor) {
	if time.Since(c.state.lastPerform) > 1500*time.Millisecond {
		c.attackFor(1300 * time.Millisecond)
	}
	if c.heavy(600 * time.Millisecond) {
		c.state.flag = true
	}
	if c.liberation() {
		c.attackFor(400 * time.Millisecond)
		c.echo()
		c.requestSwitch()
		return
	}
	if c.skill() {
		c.state.flag = false
		c.requestSwitch()
		return
	}
	c.echo()
	c.attackFor(310 * time.Millisecond)
	c.requestSwitch()
}

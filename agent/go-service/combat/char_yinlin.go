package combat

import "time"

func performYinlin(c combatActor) {
	if time.Since(c.state.lastPerform) > 1500*time.Millisecond {
		c.sleep(400 * time.Millisecond)
	}
	if c.liberation() {
		c.heavy(300 * time.Millisecond)
		c.requestSwitch()
		return
	}
	if c.skill() {
		c.sleep(120 * time.Millisecond)
		c.requestSwitch()
		return
	}
	if !c.echo() {
		c.heavy(350 * time.Millisecond)
	}
	c.requestSwitch()
}

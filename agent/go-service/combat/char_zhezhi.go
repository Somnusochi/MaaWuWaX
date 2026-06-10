package combat

import "time"

func performZhezhi(c combatActor) {
	if time.Since(c.state.lastPerform) > 2*time.Second {
		c.attackFor(1500 * time.Millisecond)
	}
	c.liberation()
	if c.state.flag && c.skill() {
		deadline := time.Now().Add(1800 * time.Millisecond)
		for time.Now().Before(deadline) {
			c.attack()
			c.sleep(90 * time.Millisecond)
		}
		c.state.flag = false
		c.requestSwitch()
		return
	}
	if c.skill() {
		c.attackFor(800 * time.Millisecond)
		c.state.flag = true
		c.requestSwitch()
		return
	}
	if !c.echo() {
		c.attack()
	}
	c.requestSwitch()
}

package combat

import "time"

func performRoccia(c combatActor) {
	if time.Since(c.state.lastPerform) > 1500*time.Millisecond {
		c.heavy(1600 * time.Millisecond)
		c.forwardAttackFor(700 * time.Millisecond)
	}
	liberated := c.liberation()
	if c.skill() || !liberated {
		c.forwardAttackFor(900 * time.Millisecond)
		c.requestSwitch()
		return
	}
	c.echo()
	c.requestSwitch()
}

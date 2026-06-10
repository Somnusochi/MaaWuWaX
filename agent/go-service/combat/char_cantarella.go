package combat

import "time"

func performCantarella(c combatActor) {
	if time.Since(c.state.lastPerform) > 1500*time.Millisecond {
		c.attackFor(1200 * time.Millisecond)
	}
	c.liberation()
	if c.skill() {
		c.heavy(700 * time.Millisecond)
		c.state.lastHeavy = time.Now()
		c.requestSwitch()
		return
	}
	if time.Since(c.state.lastHeavy) < 8*time.Second {
		deadline := time.Now().Add(1200 * time.Millisecond)
		for time.Now().Before(deadline) {
			if c.skill() {
				c.requestSwitch()
				return
			}
			c.attack()
			c.sleep(90 * time.Millisecond)
		}
	}
	if !c.echo() {
		c.attackFor(100 * time.Millisecond)
	}
	c.requestSwitch()
}

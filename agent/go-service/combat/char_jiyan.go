package combat

import "time"

func performJiyan(c combatActor) {
	if time.Since(c.state.lastPerform) > 1500*time.Millisecond {
		c.attackFor(2 * time.Second)
	}
	if c.liberation() {
		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) {
			c.skill()
			c.attack()
			c.sleep(80 * time.Millisecond)
		}
		c.requestSwitch()
		return
	}
	c.heavy(350 * time.Millisecond)
	if c.skill() {
		c.sleep(200 * time.Millisecond)
	}
	c.echo()
	c.requestSwitch()
}

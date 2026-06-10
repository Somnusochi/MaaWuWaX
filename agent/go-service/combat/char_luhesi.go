package combat

import "time"

func performLuhesi(c combatActor) {
	if time.Since(c.state.lastPerform) > 1500*time.Millisecond {
		c.attackFor(1100 * time.Millisecond)
	}
	if c.slot.Role != roleSub {
		c.skill()
		c.requestSwitch()
		return
	}
	c.jumpAttackFor(400 * time.Millisecond)
	c.heavy(600 * time.Millisecond)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if c.skill() {
			c.attackFor(180 * time.Millisecond)
		} else {
			c.attack()
		}
		if c.liberation() {
			break
		}
		c.sleep(80 * time.Millisecond)
	}
	c.echo()
	c.requestSwitch()
}

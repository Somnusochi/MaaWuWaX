package combat

import "time"

func performQiuyuan(c combatActor) {
	if time.Since(c.state.lastPerform) > 1500*time.Millisecond {
		c.attackFor(1170 * time.Millisecond)
	}
	deadline := time.Now().Add(1200 * time.Millisecond)
	if c.slot.Role == roleSub {
		deadline = time.Now().Add(4 * time.Second)
	}
	for time.Now().Before(deadline) {
		c.echo()
		c.liberation()
		if c.heavy(220 * time.Millisecond) {
			c.sleep(70 * time.Millisecond)
		}
		c.attack()
	}
	c.skill()
	c.requestSwitch()
}

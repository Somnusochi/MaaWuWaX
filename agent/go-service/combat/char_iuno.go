package combat

import "time"

func performIuno(c combatActor) {
	c.echo()
	if time.Since(c.state.lastHeavy) > 20*time.Second {
		c.heavy(900 * time.Millisecond)
		c.state.lastHeavy = time.Now()
		c.requestSwitch()
		return
	}
	if c.liberation() {
		c.attackFor(600 * time.Millisecond)
	}
	deadline := time.Now().Add(1500 * time.Millisecond)
	lastSkill := false
	for time.Now().Before(deadline) {
		if lastSkill {
			c.attack()
		} else {
			c.skill()
		}
		lastSkill = !lastSkill
		c.sleep(100 * time.Millisecond)
	}
	c.requestSwitch()
}

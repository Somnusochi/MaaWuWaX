package combat

import "time"

func performLinnai(c combatActor) {
	if !c.state.flag {
		c.attackFor(1330 * time.Millisecond)
		c.echo()
		if !c.liberation() {
			c.skill()
		}
		c.heavy(1200 * time.Millisecond)
		c.sleep(400 * time.Millisecond)
		c.jumpAttackFor(500 * time.Millisecond)
		c.skill()
		c.state.flag = true
		c.state.phaseUntil = time.Now().Add(3 * time.Second)
		c.requestSwitch()
		return
	}

	if time.Now().Before(c.state.phaseUntil) {
		c.jumpAttackFor(400 * time.Millisecond)
		if c.skill() {
			c.sleep(300 * time.Millisecond)
		}
	}
	if c.liberation() {
		c.attackFor(500 * time.Millisecond)
	}
	c.echo()
	c.requestSwitch()
}

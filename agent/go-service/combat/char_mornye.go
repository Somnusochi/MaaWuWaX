package combat

import "time"

func performMornye(c combatActor) {
	if !c.state.flag {
		c.attackFor(1330 * time.Millisecond)
		c.state.flag = true
	}

	if time.Since(c.state.lastHeavy) < 23*time.Second {
		if c.echo() {
			c.requestSwitch()
			return
		}
		if c.skill() {
			c.requestSwitch()
			return
		}
		c.attackFor(150 * time.Millisecond)
		c.requestSwitch()
		return
	}

	if c.heavy(700 * time.Millisecond) {
		c.state.lastHeavy = time.Now()
		c.state.phaseUntil = time.Now().Add(2500 * time.Millisecond)
	}
	for time.Now().Before(c.state.phaseUntil) {
		c.liberation()
		c.skill()
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	c.requestSwitch()
}

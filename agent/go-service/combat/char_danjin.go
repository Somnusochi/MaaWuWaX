package combat

import "time"

func performDanjin(c combatActor) {
	if c.liberation() {
		c.sleep(1200 * time.Millisecond)
		c.echo()
		c.requestSwitch()
		return
	}
	if !c.state.flag && time.Since(c.state.lastHeavy) > 6*time.Second {
		c.heavy(800 * time.Millisecond)
		c.state.lastHeavy = time.Now()
		c.attackFor(100 * time.Millisecond)
		c.requestSwitch()
		return
	}
	if !c.state.flag {
		c.attackFor(1100 * time.Millisecond)
		c.state.flag = true
	} else {
		c.attackFor(400 * time.Millisecond)
	}
	c.skill()
	c.attackFor(450 * time.Millisecond)
	c.skill()
	c.requestSwitch()
}

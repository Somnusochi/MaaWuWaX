package combat

import "time"

func performLupa(c combatActor) {
	if !c.state.flag {
		c.attackFor(1 * time.Second)
		c.state.flag = true
	}
	c.echo()
	if c.skill() {
		c.attackFor(300 * time.Millisecond)
	}
	c.heavy(1200 * time.Millisecond)
	if c.liberation() {
		c.state.lastLiberation = time.Now()
		c.attackFor(300 * time.Millisecond)
	}
	if time.Since(c.state.lastLiberation) < 12*time.Second {
		c.jumpAttackFor(1500 * time.Millisecond)
		c.heavy(1200 * time.Millisecond)
		c.requestSwitch()
		return
	}
	c.attackFor(100 * time.Millisecond)
	c.requestSwitch()
}

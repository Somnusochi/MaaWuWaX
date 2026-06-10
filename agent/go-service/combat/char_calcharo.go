package combat

import "time"

func performCalcharo(c combatActor) {
	if c.slot.Role != roleMain {
		c.attackFor(500 * time.Millisecond)
		c.echo()
		c.requestSwitch()
		return
	}

	if time.Since(c.state.lastPerform) > 8*time.Second {
		c.state.flag = false
	}
	if !c.state.flag {
		c.attackFor(1 * time.Second)
		c.state.flag = true
	}

	c.echo()
	if c.liberation() {
		c.attackFor(900 * time.Millisecond)
	}
	if !c.skill() {
		c.heavy(600 * time.Millisecond)
	}
	c.attackFor(1100 * time.Millisecond)
	c.requestSwitch()
}

package combat

import "time"

func performGalbrena(c combatActor) {
	if !c.state.flag {
		c.heavy(1440 * time.Millisecond)
		c.forwardAttackFor(600 * time.Millisecond)
		c.state.flag = true
	}

	c.echo()
	if c.skill() {
		c.state.phaseUntil = time.Now().Add(10 * time.Second)
	}
	if c.liberation() {
		c.attackFor(1 * time.Second)
	}
	if !c.state.phaseUntil.IsZero() && time.Now().Before(c.state.phaseUntil) {
		c.attackFor(1 * time.Second)
	}
	c.attackFor(1 * time.Second)
	c.skill()
	c.requestSwitch()
}

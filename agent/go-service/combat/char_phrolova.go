package combat

import "time"

func performPhrolova(c combatActor) {
	if !c.state.flag {
		c.attackFor(1700 * time.Millisecond)
		c.heavy(100 * time.Millisecond)
		c.state.flag = true
	}
	if c.liberation() {
		c.requestSwitch()
		return
	}
	if c.heavy(600 * time.Millisecond) {
		c.state.phaseUntil = time.Now().Add(3 * time.Second)
	}
	if c.skill() {
		c.attackFor(400 * time.Millisecond)
	}
	if time.Now().Before(c.state.phaseUntil) {
		c.attackFor(1 * time.Second)
	}
	if !c.echo() {
		c.heavy(100 * time.Millisecond)
	}
	c.requestSwitch()
}

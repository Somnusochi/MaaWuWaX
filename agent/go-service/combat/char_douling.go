package combat

import "time"

func performDouling(c combatActor) {
	if !c.state.flag {
		c.attackFor(1200 * time.Millisecond)
		c.state.flag = true
	}
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if c.liberation() {
			c.sleep(1 * time.Millisecond)
			continue
		}
		if c.skill() {
			c.sleep(1 * time.Millisecond)
			continue
		}
		c.attack()
		c.sleep(80 * time.Millisecond)
	}
	c.echo()
	c.requestSwitch()
}

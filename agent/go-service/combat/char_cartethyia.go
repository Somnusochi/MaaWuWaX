package combat

import "time"

func performCartethyia(c combatActor) {
	if time.Since(c.state.lastPerform) > 10*time.Second {
		c.state.flag = false
		c.state.phaseUntil = time.Time{}
	}

	if !c.state.flag {
		c.attackFor(1200 * time.Millisecond)
		c.echo()
		if c.skill() {
			c.attackFor(700 * time.Millisecond)
		}
		if c.liberation() {
			c.state.flag = true
			c.state.phaseUntil = time.Now().Add(3300 * time.Millisecond)
		}
	}

	if c.state.flag {
		window := 3250 * time.Millisecond
		if !c.state.phaseUntil.IsZero() {
			window = time.Until(c.state.phaseUntil)
			if window < 1200*time.Millisecond {
				window = 1200 * time.Millisecond
			}
		}
		if c.skill() {
			c.attackFor(500 * time.Millisecond)
		}
		c.attackFor(window)
		if c.liberation() {
			c.state.flag = false
		}
		c.requestSwitch()
		return
	}

	c.attackFor(1500 * time.Millisecond)
	c.requestSwitch()
}

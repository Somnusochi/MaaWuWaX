package combat

import "time"

func performJinhsi(c combatActor) {
	now := time.Now()

	// Approximate ok-ww's intro/incarnation cycle with lightweight local state.
	if c.state.flag {
		// Incarnation follow-up: alternate skill and attacks for a short burst,
		// then spend echo before leaving.
		deadline := c.state.phaseUntil
		if deadline.IsZero() || now.After(deadline) {
			deadline = now.Add(3 * time.Second)
		}
		for time.Now().Before(deadline) {
			if c.skill() {
				c.attackFor(250 * time.Millisecond)
			} else {
				c.attackFor(220 * time.Millisecond)
			}
			c.sleep(80 * time.Millisecond)
		}
		c.echo()
		c.state.flag = false
		c.state.flag2 = false
		c.requestSwitch()
		return
	}

	if c.skill() {
		// Resonance entry creates the next incarnation window.
		c.attackFor(550 * time.Millisecond)
		if c.skill() {
			c.attackFor(300 * time.Millisecond)
		}
		c.state.flag = true
		c.state.phaseUntil = time.Now().Add(3200 * time.Millisecond)
		c.requestSwitch()
		return
	}

	if c.liberation() {
		c.attackFor(1200 * time.Millisecond)
		c.state.flag = true
		c.state.phaseUntil = time.Now().Add(2800 * time.Millisecond)
		c.requestSwitch()
		return
	}

	c.echo()
	c.requestSwitch()
}

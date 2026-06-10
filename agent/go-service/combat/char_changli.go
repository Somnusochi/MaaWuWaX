package combat

import "time"

func performChangli(c combatActor) {
	// Carry a short-lived "enhanced normal" phase across switches.
	if c.state.flag {
		c.attackFor(400 * time.Millisecond)
		c.sleep(180 * time.Millisecond)
		c.state.flag = false
	}

	// Full forte in ok-ww becomes heavy-first priority; we approximate with
	// "recent liberation" and a longer heavy release.
	if time.Since(c.state.lastLiberation) < 2*time.Second {
		c.heavy(900 * time.Millisecond)
		c.requestSwitch()
		return
	}

	if c.liberation() {
		c.heavy(700 * time.Millisecond)
		c.requestSwitch()
		return
	}

	if c.skill() {
		c.state.flag = true
		c.attackFor(260 * time.Millisecond)
		c.requestSwitch()
		return
	}

	if c.echo() {
		c.requestSwitch()
		return
	}

	c.attackFor(250 * time.Millisecond)
	c.requestSwitch()
}

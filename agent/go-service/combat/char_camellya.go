package combat

import "time"

func performCamellya(c combatActor) {
	// Intro in ok-ww spends a long heavy sequence immediately.
	if time.Since(c.state.lastPerform) > 1500*time.Millisecond {
		c.attackFor(1200 * time.Millisecond)
		c.heavy(2200 * time.Millisecond)
	}

	if c.liberation() {
		c.attackFor(500 * time.Millisecond)
	}

	// Use resonance to enter the short build phase when concerto is not full.
	if c.skill() {
		deadline := time.Now().Add(1200 * time.Millisecond)
		for time.Now().Before(deadline) {
			c.attack()
			c.sleep(90 * time.Millisecond)
		}
	} else {
		// Fuller budding-style window approximation.
		c.heavy(1800 * time.Millisecond)
	}

	c.echo()
	c.requestSwitch()
}

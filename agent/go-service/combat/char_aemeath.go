package combat

import "time"

func performAemeath(c combatActor) {
	if time.Since(c.state.lastPerform) > 10*time.Second {
		c.state.flag = false
		c.state.phaseUntil = time.Time{}
		c.state.introTime = 0
	}

	if !c.state.flag {
		c.attackFor(1200 * time.Millisecond)
		c.state.flag = true
		if c.state.introTime == 0 {
			c.state.introTime = 14 * time.Second
		}
	}

	start := time.Now()
	waitLonger := time.Since(c.state.lastLiberation) > 17*time.Second || time.Since(c.state.lastResonance) > 12*time.Second
	limit := 1200 * time.Millisecond
	if waitLonger {
		limit = 3600 * time.Millisecond
	}
	for time.Since(start) < limit {
		if !c.state.phaseUntil.IsZero() && time.Now().Before(c.state.phaseUntil) {
			c.heavy(400 * time.Millisecond)
			c.state.phaseUntil = time.Time{}
			continue
		}
		if c.liberation() {
			c.state.lastLiberation = time.Now()
			c.state.phaseUntil = time.Now().Add(25 * time.Second)
			c.heavy(200 * time.Millisecond)
			continue
		}
		if c.skill() {
			c.attackFor(250 * time.Millisecond)
		}
		c.echo()
		c.attackFor(300 * time.Millisecond)
	}
	c.requestSwitch()
}

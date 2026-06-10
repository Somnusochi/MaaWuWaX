package combat
import "time"
func performEncore(c combatActor) {
	if time.Since(c.state.lastLiberation) < 9*time.Second {
		c.skill(); c.attackFor(700*time.Millisecond)
		if time.Since(c.state.lastLiberation) > 6*time.Second && time.Since(c.state.lastHeavy) > 4*time.Second {
			c.heavy(650*time.Millisecond); c.state.lastHeavy = time.Now()
		}
		c.requestSwitch(); return
	}
	if c.skill() && time.Since(c.state.lastResonance) > 4*time.Second { c.state.lastResonance = time.Now(); c.requestSwitch(); return }
	if c.liberation() { c.state.lastLiberation = time.Now(); c.attackFor(900*time.Millisecond); c.requestSwitch(); return }
	c.echo(); c.requestSwitch()
}

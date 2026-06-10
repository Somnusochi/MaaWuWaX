package combat
import "time"
func performAemeath(c combatActor) {
	c.attackFor(1200*time.Millisecond)
	start := time.Now()
	for time.Since(start) < 3600*time.Millisecond {
		if c.liberation() { c.state.lastLiberation = time.Now(); c.heavy(200*time.Millisecond) }
		c.skill(); c.echo(); c.attackFor(300*time.Millisecond)
		if time.Since(start) > 1200*time.Millisecond { break }
	}
	c.requestSwitch()
}

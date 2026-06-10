package combat
import "time"
func performRebecca(c combatActor) {
	c.attackFor(1000*time.Millisecond); c.echo(); c.heavy(600*time.Millisecond)
	if c.skill() { c.attackFor(350*time.Millisecond) }
	c.heavy(600*time.Millisecond)
	if c.liberation() {
		start := time.Now()
		for time.Since(start) < 5200*time.Millisecond {
			c.attack()
			if time.Since(c.state.lastLiberation) > 900*time.Millisecond { c.liberation() }
			c.sleep(80*time.Millisecond)
		}
		c.requestSwitch(); return
	}
	c.attackFor(700*time.Millisecond); c.requestSwitch()
}

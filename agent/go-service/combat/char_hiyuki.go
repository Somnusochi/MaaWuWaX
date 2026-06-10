package combat
import "time"
func performHiyuki(c combatActor) {
	c.attackFor(1000*time.Millisecond)
	if c.liberation() { c.attackFor(300*time.Millisecond) }
	start := time.Now()
	for time.Since(start) < 6*time.Second {
		c.echo()
		if c.liberation() { break }
		c.heavy(350*time.Millisecond); c.attackFor(250*time.Millisecond); c.skill()
		if c.liberation() { break }
	}
	c.requestSwitch()
}

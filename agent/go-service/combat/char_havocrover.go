package combat
import "time"
func performHavocRover(c combatActor) {
	c.attackFor(500*time.Millisecond); c.heavy(600*time.Millisecond); c.echo()
	if c.skill() { c.attackFor(1400*time.Millisecond) }
	c.liberation(); c.requestSwitch()
}

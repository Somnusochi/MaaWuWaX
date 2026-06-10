package combat
import "time"
func performDenia(c combatActor) {
	c.attackFor(450*time.Millisecond); c.skill()
	if c.liberation() { c.skill() }; c.echo(); c.requestSwitch()
}

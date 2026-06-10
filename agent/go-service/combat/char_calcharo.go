package combat
import "time"
func performCalcharo(c combatActor) {
	c.attackFor(700*time.Millisecond); c.echo(); c.liberation()
	if !c.skill() { c.heavy(600*time.Millisecond) }; c.requestSwitch()
}

package combat
import "time"
func performLuhesi(c combatActor) {
	c.attackFor(1100*time.Millisecond); c.jumpAttackFor(400*time.Millisecond); c.heavy(600*time.Millisecond)
	if c.skill() { c.liberation() }
	c.echo(); c.requestSwitch()
}

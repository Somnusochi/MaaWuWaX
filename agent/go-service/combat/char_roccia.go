package combat
import "time"
func performRoccia(c combatActor) {
	c.heavy(900*time.Millisecond); liberated := c.liberation()
	if c.skill() || !liberated { c.forwardAttackFor(900*time.Millisecond); c.requestSwitch(); return }
	c.echo(); c.requestSwitch()
}

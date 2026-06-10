package combat
import "time"
func performLupa(c combatActor) {
	c.attackFor(1000*time.Millisecond); c.echo()
	if c.skill() { c.attackFor(300*time.Millisecond) }
	c.heavy(900*time.Millisecond)
	if c.liberation() { c.attackFor(1200*time.Millisecond); c.jumpAttackFor(500*time.Millisecond); c.heavy(800*time.Millisecond) }
	c.requestSwitch()
}

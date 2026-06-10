package combat
import "time"
func performJiyan(c combatActor) {
	c.attackFor(800*time.Millisecond)
	if c.liberation() { c.attackFor(1200*time.Millisecond); c.skill(); c.attackFor(800*time.Millisecond); c.requestSwitch(); return }
	c.heavy(350*time.Millisecond); c.skill(); c.echo(); c.requestSwitch()
}

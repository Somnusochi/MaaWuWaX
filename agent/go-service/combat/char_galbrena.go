package combat
import "time"
func performGalbrena(c combatActor) {
	c.heavy(550*time.Millisecond); c.echo(); c.skill()
	if c.liberation() { c.attackFor(900*time.Millisecond) }; c.attackFor(500*time.Millisecond); c.requestSwitch()
}

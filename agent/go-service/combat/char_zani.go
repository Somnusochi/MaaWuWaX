package combat
import "time"
func performZani(c combatActor) {
	c.attackFor(1300*time.Millisecond); c.echo(); c.skill(); c.liberation(); c.attackFor(500*time.Millisecond); c.heavy(600*time.Millisecond); c.requestSwitch()
}

package combat
import "time"
func performPhoebe(c combatActor) {
	c.attackFor(1500*time.Millisecond); c.echo(); c.liberation(); c.heavy(500*time.Millisecond); c.skill(); c.echo(); c.requestSwitch()
}

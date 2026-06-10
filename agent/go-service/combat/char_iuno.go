package combat
import "time"
func performIuno(c combatActor) {
	c.echo(); c.heavy(600*time.Millisecond); c.liberation(); c.skill(); c.attackFor(500*time.Millisecond); c.requestSwitch()
}

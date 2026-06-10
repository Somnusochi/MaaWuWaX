package combat
import "time"
func performCiaccona(c combatActor) {
	c.attackFor(1500*time.Millisecond); c.echo(); c.skill(); c.heavy(600*time.Millisecond); c.liberation(); c.requestSwitch()
}

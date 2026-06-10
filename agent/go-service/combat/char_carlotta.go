package combat
import "time"
func performCarlotta(c combatActor) {
	c.attackFor(1300*time.Millisecond); c.heavy(600*time.Millisecond); c.liberation(); c.skill(); c.echo(); c.requestSwitch()
}

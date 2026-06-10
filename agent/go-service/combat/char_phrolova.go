package combat
import "time"
func performPhrolova(c combatActor) {
	c.attackFor(1700*time.Millisecond); c.liberation(); c.heavy(600*time.Millisecond); c.skill(); c.echo(); c.requestSwitch()
}

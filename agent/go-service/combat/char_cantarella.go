package combat
import "time"
func performCantarella(c combatActor) {
	c.attackFor(1200*time.Millisecond); c.liberation(); c.skill(); c.heavy(600*time.Millisecond); c.echo(); c.requestSwitch()
}

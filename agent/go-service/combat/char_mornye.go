package combat
import "time"
func performMornye(c combatActor) {
	c.attackFor(1330*time.Millisecond); c.echo(); c.skill(); c.heavy(600*time.Millisecond); c.liberation(); c.requestSwitch()
}

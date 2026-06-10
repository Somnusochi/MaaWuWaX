package combat
import "time"
func performCamellya(c combatActor) {
	c.attackFor(1200*time.Millisecond); c.liberation(); c.skill(); c.heavy(4600*time.Millisecond); c.echo(); c.requestSwitch()
}

package combat
import "time"
func performShorekeeper(c combatActor) {
	c.attackFor(500*time.Millisecond); c.echo(); c.liberation(); c.skill(); c.requestSwitch()
}

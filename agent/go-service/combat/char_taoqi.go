package combat
import "time"
func performTaoqi(c combatActor) {
	c.attackFor(900*time.Millisecond); c.liberation(); c.skill(); c.echo(); c.requestSwitch()
}

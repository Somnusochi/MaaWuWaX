package combat
import "time"
func performBaizhi(c combatActor) {
	c.attackFor(700*time.Millisecond); c.liberation(); c.skill(); c.echo(); c.requestSwitch()
}

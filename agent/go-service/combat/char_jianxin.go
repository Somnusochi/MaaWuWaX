package combat
import "time"
func performJianxin(c combatActor) {
	c.attackFor(500*time.Millisecond); c.liberation(); c.skill(); c.echo(); c.requestSwitch()
}

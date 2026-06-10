package combat
import "time"
func performJinhsi(c combatActor) {
	c.echo(); c.skill(); c.attackFor(550*time.Millisecond); c.skill(); c.requestSwitch()
}

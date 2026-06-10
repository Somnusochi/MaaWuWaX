package combat
import "time"
func performCartethyia(c combatActor) {
	c.attackFor(1200*time.Millisecond); c.liberation(); c.skill(); c.attackFor(3250*time.Millisecond); c.requestSwitch()
}

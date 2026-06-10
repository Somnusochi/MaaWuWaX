package combat
import "time"
func performChisa(c combatActor) {
	if c.slot.Role == roleHealer {
		c.echo()
		if c.liberation() { c.state.lastBuff = time.Now() }
		c.skill(); c.requestSwitch(); return
	}
	c.attackFor(800*time.Millisecond); c.echo(); c.heavy(600*time.Millisecond); c.liberation(); c.skill(); c.requestSwitch()
}

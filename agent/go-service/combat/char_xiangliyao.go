package combat
import "time"
func performXiangliyao(c combatActor) {
	c.attackFor(300*time.Millisecond)
	if c.liberation() { c.state.lastLiberation = time.Now() }
	if time.Since(c.state.lastLiberation) < 25*time.Second { c.skill(); c.attackFor(600*time.Millisecond) } else if !c.echo() { c.skill() }
	c.requestSwitch()
}

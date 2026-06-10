package combat
import "time"
func performZhezhi(c combatActor) {
	if time.Since(c.state.lastPerform) > 2*time.Second { c.attackFor(700*time.Millisecond) }
	c.liberation()
	if c.skill() { c.attackFor(500*time.Millisecond); c.state.flag = !c.state.flag }
	if !c.echo() { c.attack() }; c.requestSwitch()
}

package combat
import "time"
func performYinlin(c combatActor) {
	if c.liberation() { c.heavy(250*time.Millisecond); c.requestSwitch(); return }
	if c.skill() { c.sleep(120*time.Millisecond); c.requestSwitch(); return }
	if !c.echo() { c.heavy(300*time.Millisecond) }; c.requestSwitch()
}

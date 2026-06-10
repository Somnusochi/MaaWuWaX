package combat
import "time"
func performDanjin(c combatActor) {
	if c.liberation() { c.sleep(800*time.Millisecond); c.echo(); c.requestSwitch(); return }
	c.attackFor(450*time.Millisecond); c.skill(); c.attackFor(450*time.Millisecond); c.skill(); c.requestSwitch()
}

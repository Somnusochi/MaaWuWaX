package combat
import "time"
func performQiuyuan(c combatActor) {
	deadline := time.Now().Add(1200*time.Millisecond)
	for time.Now().Before(deadline) { c.echo(); c.liberation(); c.attack() }
	c.skill(); c.requestSwitch()
}

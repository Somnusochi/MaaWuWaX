package combat
import "time"
func performDouling(c combatActor) {
	deadline := time.Now().Add(1*time.Second)
	for time.Now().Before(deadline) { if c.liberation() { continue }; c.skill(); c.attack() }
	c.echo(); c.requestSwitch()
}

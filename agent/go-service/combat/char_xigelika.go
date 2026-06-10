package combat
import "time"
func performXigelika(c combatActor) {
	c.attackFor(770*time.Millisecond)
	start := time.Now()
	baseTimeout := 500 * time.Millisecond
	if c.slot.Role == roleSub { baseTimeout = 15 * time.Second }
	deadline := start.Add(baseTimeout)
	for time.Now().Before(deadline) {
		c.echo(); c.heavy(600*time.Millisecond)
		if c.liberation() { deadline = time.Now().Add(15 * time.Second) }
		c.skill(); c.attack(); c.sleep(50*time.Millisecond)
	}
	c.requestSwitch()
}

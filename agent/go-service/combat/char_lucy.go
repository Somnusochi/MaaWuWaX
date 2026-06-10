package combat
import "time"
func performLucy(c combatActor) {
	c.attackFor(1000*time.Millisecond); c.echo()
	if c.skill() {
		start := time.Now()
		for time.Since(start) < 16*time.Second {
			c.attack()
			if c.liberation() { c.heavy(200*time.Millisecond); c.requestSwitch(); return }
			c.sleep(90*time.Millisecond)
		}
		c.requestSwitch(); return
	}
	if c.liberation() { c.heavy(200*time.Millisecond); c.requestSwitch(); return }
	if c.skill() {
		start := time.Now()
		for time.Since(start) < 1400*time.Millisecond {
			if c.skill() { c.attack(); if c.liberation() { c.heavy(200*time.Millisecond) }; break }
			c.attack(); c.sleep(90*time.Millisecond)
		}
		c.requestSwitch(); return
	}
	c.attackFor(800*time.Millisecond); c.requestSwitch()
}

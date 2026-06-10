package combat
import "time"
func performLinnai(c combatActor) {
	c.attackFor(1330*time.Millisecond); c.echo(); c.liberation(); c.skill()
	c.heavy(600*time.Millisecond); c.jumpAttackFor(500*time.Millisecond); c.skill(); c.requestSwitch()
}

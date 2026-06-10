package combat
import "time"
func performChangli(c combatActor) {
	c.attackFor(300*time.Millisecond); c.liberation(); c.heavy(600*time.Millisecond); c.skill(); c.echo(); c.requestSwitch()
}

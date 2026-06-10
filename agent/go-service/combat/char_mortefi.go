package combat
import "time"
func performMortefi(c combatActor) {
	liberated := c.liberation(); c.skill(); c.echo()
	if !liberated { c.sleep(250*time.Millisecond); c.liberation() }; c.requestSwitch()
}

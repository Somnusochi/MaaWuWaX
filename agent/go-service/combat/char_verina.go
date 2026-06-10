package combat
func performVerina(c combatActor) {
	if c.liberation() { c.requestSwitch(); return }
	if c.skill() { c.attack(); c.requestSwitch(); return }
	c.echo(); c.requestSwitch()
}

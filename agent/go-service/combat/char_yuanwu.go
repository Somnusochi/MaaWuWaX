package combat
func performYuanwu(c combatActor) {
	if c.liberation() { c.skill(); c.requestSwitch(); return }
	c.skill(); c.echo(); c.requestSwitch()
}

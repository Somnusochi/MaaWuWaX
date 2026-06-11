package combat

// performChixia mirrors ok-ww Chixia: do_perform() is commented out in Python,
// so the character falls through to BaseChar.do_perform() (the default rotation).
// In Go this is equivalent to performDefault().
func performChixia(c combatActor) {
	performDefault(c)
}

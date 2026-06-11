package combat

// performYouhu mirrors ok-ww Youhu: class has no do_perform() override,
// so the character falls through to BaseChar.do_perform() (the default rotation).
// In Go this is equivalent to performDefault().
func performYouhu(c combatActor) {
	performDefault(c)
}

package combat

import "time"

// performJianxin mirrors ok-ww Jianxin.do_perform():
//
//	intro(1s attack) → liberation → resonance → echo → switch
func performJianxin(c combatActor) {
	if c.recentlyIntroSwitchedIn(1500 * time.Millisecond) {
		c.attackFor(1 * time.Second)
	}
	defaultClickLiberation(c)
	if c.resonanceAvailable() {
		defaultClickResonance(c)
	}
	if c.echoNoCD() {
		c.echoWait(1 * time.Second)
	}
	c.requestSwitch()
}

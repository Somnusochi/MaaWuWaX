package combat

import "time"

// performTaoqi mirrors ok-ww Taoqi.do_perform():
//
//	intro: wait_down(0.9s) + attack(2.5s) → switch / !intro: liberation → resonance → echo → switch
func performTaoqi(c combatActor) {
	if c.recentlyIntroSwitchedIn(3 * time.Second) {
		c.waitDown(900 * time.Millisecond)
		c.attackFor(2500 * time.Millisecond)
		c.requestSwitch()
		return
	}
	defaultClickLiberation(c)
	defaultClickResonance(c)
	c.echo()
	c.requestSwitch()
}

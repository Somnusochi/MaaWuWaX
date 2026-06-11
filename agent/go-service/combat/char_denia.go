package combat

import "time"

// performDenia mirrors ok-ww Denia.do_perform():
//
//	intro(wait_intro 1.2s) → resonance → liberation→resonance → echo → switch
func performDenia(c combatActor) {
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		c.waitIntro(1200*time.Millisecond, true)
	}
	if c.currentResonance() > 0.05 {
		defaultClickResonance(c)
	}
	if defaultClickLiberation(c) {
		defaultClickResonance(c)
	}
	c.echoWait(1 * time.Second)
	c.requestSwitch()
}

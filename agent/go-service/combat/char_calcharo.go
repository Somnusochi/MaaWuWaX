package combat

import "time"

// performCalcharo mirrors ok-ww Calcharo.do_perform():
//
//	intro(sleep 1s, wait_in_team 3s) → super().do_perform():
//	  wait_intro(1.2s) → echo → liberation → resonance / heavy_click_forte → switch
func performCalcharo(c combatActor) {
	intro := c.recentlyIntroSwitchedIn(1800 * time.Millisecond)
	if intro {
		c.sleep(1 * time.Second)
		c.waitIntro(3*time.Second, false)
	}
	// ok-ww: super().do_perform() = wait_intro(1.2) → echo → lib → res → heavy → switch
	if intro {
		c.waitIntro(1200*time.Millisecond, true)
	}
	c.echoImmediate()
	defaultClickLiberation(c)
	if !defaultClickResonance(c) {
		defaultHeavyClickForte(c)
	}
	c.requestSwitch()
}

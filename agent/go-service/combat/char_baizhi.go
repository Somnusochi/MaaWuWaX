package combat

import "time"

// performBaizhi mirrors ok-ww Baizhi.do_perform():
//
//	intro(1.2s attack with resonance-if-ready) → liberation(con<1) → resonance → echo → switch
func performBaizhi(c combatActor) {
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		start := time.Now()
		for time.Since(start) < 1200*time.Millisecond {
			if c.resonanceAvailable() {
				baizhiClickResonance(c)
				break
			}
			c.attack()
			c.sleep(100 * time.Millisecond)
		}
	}
	baizhiClickLiberation(c)
	if c.resonanceAvailable() {
		baizhiClickResonance(c)
	}
	if c.echoWait(1 * time.Second) {
		c.requestSwitch()
		return
	}
	c.requestSwitch()
}

// baizhiClickLiberation mirrors ok-ww Baizhi.click_liberation(con_less_than=1):
// casts liberation with up to 800ms wait, then finishLiberationCast with 3s timeout.
func baizhiClickLiberation(c combatActor) bool {
	if !c.liberationAvailable() {
		return false
	}
	start := time.Now()
	clicked := false
	for time.Since(start) < 800*time.Millisecond && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) {
		c.forceLiberation()
		clicked = true
		c.sleep(100 * time.Millisecond)
	}
	return finishLiberationCast(c, clicked, 3*time.Second)
}

// baizhiClickResonance mirrors ok-ww Baizhi.click_resonance():
// casts resonance while available for up to 15s.
func baizhiClickResonance(c combatActor) bool {
	start := time.Now()
	clicked := false
	for c.resonanceChainAvailable() && time.Since(start) < 15*time.Second {
		if c.currentResonance() > 0 && c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

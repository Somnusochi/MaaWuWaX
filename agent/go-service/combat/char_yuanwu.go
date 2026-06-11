package combat

import "time"

// performYuanwu mirrors ok-ww Yuanwu.do_perform():
//
//	liberation(con<1): resonance → switch / intro: attack(1.2s) → switch / resonance → echo → switch
func performYuanwu(c combatActor) {
	if screenAnalyzer.ConcertoPct < 1.0 && yuanwuClickLiberation(c) {
		yuanwuClickResonance(c)
		c.requestSwitch()
		return
	}
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		c.attackFor(1200 * time.Millisecond)
		c.requestSwitch()
		return
	}
	yuanwuClickResonance(c)
	c.echoImmediate()
	c.requestSwitch()
}

func yuanwuClickLiberation(c combatActor) bool {
	if !c.param.UseLiberation || screenAnalyzer.ConcertoPct >= 1.0 ||
		!c.liberationAvailable() {
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

func yuanwuClickResonance(c combatActor) bool {
	if !c.resonanceAvailable() {
		return false
	}
	start := time.Now()
	clicked := false
	for c.resonanceAvailable() && time.Since(start) < 15*time.Second {
		if c.currentResonance() > 0 && c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

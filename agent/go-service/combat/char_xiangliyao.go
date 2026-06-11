package combat

import "time"

// performXiangliyao mirrors ok-ww Xiangliyao.do_perform():
//
//	wait_down → liberation → still_in_liberation(<25s): resonance+attack_loop → echo / resonance → switch
func performXiangliyao(c combatActor) {
	c.waitDown(1500 * time.Millisecond)
	xiangliyaoClickLiberation(c)

	if !c.state.lastLiberation.IsZero() && c.freezeElapsed(c.state.lastLiberation, c.state.xiangliyaoLiberationFreeze) < 25*time.Second {
		for !xiangliyaoClickResonance(c) {
			c.attackFor(1 * time.Second)
		}
	} else if c.echoWait(1 * time.Second) {
	} else {
		xiangliyaoClickResonance(c)
	}
	c.requestSwitch()
}

func xiangliyaoClickLiberation(c combatActor) bool {
	if !c.param.UseLiberation || (!screenAnalyzer.Liberation && c.currentLiberation() <= 0.05) {
		return false
	}
	start := time.Now()
	clicked := false
	for time.Since(start) < 800*time.Millisecond && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) {
		c.forceLiberation()
		clicked = true
		c.sleep(100 * time.Millisecond)
	}
	if !finishLiberationCast(c, clicked, 3*time.Second) {
		return false
	}
	c.state.xiangliyaoLiberationFreeze = screenAnalyzer.FreezeDuration
	return true
}

func xiangliyaoClickResonance(c combatActor) bool {
	if c.currentResonance() <= 0.05 {
		return false
	}
	start := time.Now()
	clicked := false
	for c.currentResonance() > 0.05 && time.Since(start) < 15*time.Second {
		if c.forceSkill() {
			clicked = true
		}
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

package combat

import "time"

// performYinlin mirrors ok-ww Yinlin.do_perform():
//
//	intro(sleep 0.4s) → liberation → mouse_forte_full: heavy_attack → sleep(0.4s) → switch /
//	resonance → sleep(0.1s) → switch / echo → switch / heavy → switch
func performYinlin(c combatActor) {
	intro := c.recentlyIntroSwitchedIn(1600 * time.Millisecond)
	if intro {
		c.sleep(400 * time.Millisecond)
	}
	liberated := yinlinClickLiberation(c)

	if c.mouseForteFull() {
		if !intro && !liberated {
			c.attack()
		}
		yinlinHeavyAttack(c)
		c.sleep(400 * time.Millisecond)
		c.requestSwitch()
		return
	}
	if yinlinClickResonance(c) {
		c.sleep(100 * time.Millisecond)
		c.requestSwitch()
		return
	}
	if c.echoWait(1 * time.Second) {
		c.requestSwitch()
		return
	}
	yinlinHeavyAttack(c)
	c.requestSwitch()
}

func yinlinClickLiberation(c combatActor) bool {
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

func yinlinClickResonance(c combatActor) bool {
	if !c.resonanceAvailable() {
		return false
	}
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

func yinlinHeavyAttack(c combatActor) {
	c.heavy(600 * time.Millisecond)
	c.sleep(10 * time.Millisecond)
}

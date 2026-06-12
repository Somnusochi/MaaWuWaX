package combat

import "time"

// performShorekeeper mirrors ok-ww ShoreKeeper.do_perform():
//
//	intro(sleep 0.1s, wait_in_team_and_world 4s if needed / attack 1.2s if already in world)
//	→ echo → liberation → resonance(fallback: heavy_click_forte until mouse_forte_full) → switch
func performShorekeeper(c combatActor) {
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		c.sleep(100 * time.Millisecond)
		if !c.isCurrentChar() {
			c.waitIntro(4*time.Second, false)
		} else {
			c.attackFor(1200 * time.Millisecond)
		}
	}
	c.echoImmediate()
	shorekeeperClickLiberation(c)
	if !shorekeeperClickResonance(c) {
		defaultHeavyClickForte(c)
	}
	shorekeeperPrepareOutro(c)
	c.requestSwitch()
}

func shorekeeperClickLiberation(c combatActor) bool {
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

func shorekeeperClickResonance(c combatActor) bool {
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

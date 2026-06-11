package combat

import "time"

// performChisa mirrors ok-ww Chisa.do_perform():
//
//	non-DPS→do_fast_support: intro(record_buff, echo, switch) / wait_down / echo / liberation(record_buff) / resonance(0.5s)→switch
//	DPS→do_dps_perform: intro(0.8s attack, timeout=2.3) / wait_down / echo /
//	  liberation+resonance loop(0.5s window each) / forte_full→perform_forte → switch
func performChisa(c combatActor) {
	if !c.chisaDPS() {
		if c.recentlyIntroSwitchedIn(1700 * time.Millisecond) {
			c.state.lastBuff = time.Now()
			c.state.lastBuffFreeze = screenAnalyzer.FreezeDuration
			c.echo()
			c.requestSwitch()
			return
		}
		if c.flying() && !screenAnalyzer.Liberation && c.currentLiberation() <= 0.05 && c.currentResonance() <= 0.05 {
			c.waitDown(1200 * time.Millisecond)
		}
		c.echo()
		if chisaClickLiberation(c) {
			c.state.lastBuff = time.Now()
			c.state.lastBuffFreeze = screenAnalyzer.FreezeDuration
			c.requestSwitch()
			return
		}
		chisaClickResonance(c, 500*time.Millisecond)
		c.requestSwitch()
		return
	}
	timeout := 2500 * time.Millisecond
	if c.recentlyIntroSwitchedIn(1700 * time.Millisecond) {
		c.attackFor(800 * time.Millisecond)
		timeout = 2300 * time.Millisecond
	}
	if c.flying() && !screenAnalyzer.Liberation && c.currentLiberation() <= 0.05 && c.currentResonance() <= 0.05 {
		c.waitDown(1200 * time.Millisecond)
	}
	c.echoWait(1 * time.Second)
	start := time.Now()
	underLiberation := false
	for time.Since(start) < timeout {
		if time.Since(start) < 500*time.Millisecond && chisaClickLiberation(c) {
			start = time.Now()
			underLiberation = true
			timeout = 10 * time.Second
			c.sleep(200 * time.Millisecond)
		}
		if time.Since(start) < 500*time.Millisecond && !c.mouseForteFull() && chisaClickResonance(c, 15*time.Second) {
			start = time.Now()
			if !underLiberation {
				timeout = 1700 * time.Millisecond
			}
		}
		if (underLiberation || c.chisaDPS()) && c.forteFull() && chisaPerformForte(c) {
			c.requestSwitch()
			return
		}
		c.attack()
		c.sleep(90 * time.Millisecond)
	}
	c.requestSwitch()
}

// chisaClickLiberation mirrors ok-ww Chisa.click_liberation():
// standard liberation cast with finishLiberationCast.
func chisaClickLiberation(c combatActor) bool {
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
	return finishLiberationCast(c, clicked, 3*time.Second)
}

// chisaClickResonance mirrors ok-ww Chisa.click_resonance():
// casts resonance while available for the given timeout.
func chisaClickResonance(c combatActor, timeout time.Duration) bool {
	if c.currentResonance() <= 0.05 {
		return false
	}
	start := time.Now()
	clicked := false
	for c.currentResonance() > 0.05 && time.Since(start) < timeout {
		if c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

// chisaPerformForte mirrors ok-ww Chisa.perform_forte():
// flying→wait_down → holdSkill for 1.2s → if still forte_full: heavy 3.5s.
func chisaPerformForte(c combatActor) bool {
	if c.flying() {
		c.waitDown(1200 * time.Millisecond)
	}
	c.holdSkillUntil(1200*time.Millisecond, 100*time.Millisecond, func() bool {
		return !c.forteFull()
	})
	if c.forteFull() {
		return false
	}
	c.heavy(3500 * time.Millisecond)
	return true
}

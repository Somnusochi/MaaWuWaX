package combat

import "time"

// performVerina mirrors ok-ww Verina.do_perform() (do_cycle):
//
//	intro(1s attack) → loop(timeout=1s/0.4s until con_full):
//	  liberation → resonance → echo → heavy(if can_heavy_attack every 8s) → normal attack → switch
func performVerina(c combatActor) {
	intro := c.recentlyIntroSwitchedIn(1500 * time.Millisecond)
	if intro {
		c.attackFor(1 * time.Second)
	}

	start := time.Now()
	timeout := 1 * time.Second
	if !intro {
		timeout = 400 * time.Millisecond
	}
	for time.Since(start) < timeout {
		if screenAnalyzer.ConcertoPct >= 1.0 {
			break
		}
		if verinaClickLiberation(c) || verinaClickResonance(c) || c.echo() {
			c.sleep(100 * time.Millisecond)
			continue
		}
		if c.mouseForteFull() && c.freezeElapsed(c.state.lastHeavy, c.state.verinaHeavyFreeze) >= 8*time.Second {
			c.heavy(700 * time.Millisecond)
			c.state.lastHeavy = time.Now()
			c.state.verinaHeavyFreeze = screenAnalyzer.FreezeDuration
			continue
		}
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	c.requestSwitch()
}

func verinaClickLiberation(c combatActor) bool {
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

func verinaClickResonance(c combatActor) bool {
	if c.currentResonance() <= 0.05 {
		return false
	}
	start := time.Now()
	clicked := false
	for c.currentResonance() > 0.05 && time.Since(start) < 800*time.Millisecond {
		if c.forceSkill() {
			clicked = true
		}
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

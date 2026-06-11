package combat

import "time"

// performRebecca mirrors ok-ww Rebecca.do_perform():
//
//	intro(1s attack) / flying→wait_down → echo → enhanced_heavy →
//	resonance→normal_attack → enhanced_heavy → !fast+liberation→hmg_mode(5.2s) → switch
func performRebecca(c combatActor) {
	if c.recentlyIntroSwitchedIn(1700 * time.Millisecond) {
		c.attackFor(1 * time.Second)
	} else if c.flying() {
		c.waitDown(1200 * time.Millisecond)
	}
	c.echo()

	rebeccaPerformEnhancedHeavy(c)

	if rebeccaClickResonance(c) {
		c.attackFor(350 * time.Millisecond)
	}

	rebeccaPerformEnhancedHeavy(c)

	if !c.needFastPerform() && rebeccaClickLiberation(c) {
		start := time.Now()
		startFreeze := screenAnalyzer.FreezeDuration
		lastLiberation := start
		for c.freezeElapsed(start, startFreeze) < 5200*time.Millisecond {
			if c.needFastPerform() {
				break
			}
			c.attack()
			now := time.Now()
			if now.Sub(lastLiberation) > 900*time.Millisecond {
				c.forceLiberation()
				lastLiberation = now
			}
			c.sleep(80 * time.Millisecond)
		}
		c.requestSwitch()
		return
	}
	c.attackFor(700 * time.Millisecond)
	c.requestSwitch()
}

func rebeccaClickLiberation(c combatActor) bool {
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

// rebeccaPerformEnhancedHeavy mirrors ok-ww Rebecca.perform_enhanced_heavy():
// heavy_click_forte → normal_attack(0.25s).
func rebeccaPerformEnhancedHeavy(c combatActor) bool {
	if !c.forteFull() {
		return false
	}
	c.holdHeavyUntil(2*time.Second, 100*time.Millisecond, func() bool {
		return !c.forteFull()
	})
	if c.forteFull() {
		return false
	}
	c.attackFor(250 * time.Millisecond)
	return true
}

// rebeccaClickResonance mirrors ok-ww Rebecca.click_resonance(time_out=0.8):
// casts resonance with 0.8s timeout to avoid long stalls.
func rebeccaClickResonance(c combatActor) bool {
	start := time.Now()
	clicked := false
	for c.resonanceAvailable() && time.Since(start) < 800*time.Millisecond {
		if c.currentResonance() > 0 && c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

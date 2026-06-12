package combat

import "time"

// performLucy mirrors ok-ww Lucy.do_perform():
//
//	intro(1s attack) / flying→wait_down → echo →
//	forte_full→resonance→algorithm_compaction(5s loop: multi_threading / liberation / click) → switch /
//	!fast+liberation→switch / resonance→1.4s forte_wait→algorithm / normal_attack(0.8s)→switch
func performLucy(c combatActor) {
	if c.recentlyIntroSwitchedIn(1500 * time.Millisecond) {
		c.attackFor(1000 * time.Millisecond)
	} else if c.flying() {
		c.waitDown(1200 * time.Millisecond)
	}

	c.echo()

	if c.forteFull() && lucyClickResonance(c) {
		c.state.lucyAlgorithmUntil = time.Now()
		c.state.lucyAlgorithmFreeze = screenAnalyzer.FreezeDuration
		lucyAlgorithmCompaction(c)
		c.requestSwitch()
		return
	}

	if !c.needFastPerform() && lucyClickLiberation(c) {
		c.fBreak()
		c.requestSwitch()
		return
	}

	if lucyClickResonance(c) {
		start := time.Now()
		startFreeze := screenAnalyzer.FreezeDuration
		for c.freezeElapsed(start, startFreeze) < 1400*time.Millisecond {
			if c.forteFull() {
				if lucyClickResonance(c) {
					c.state.lucyAlgorithmUntil = time.Now()
					c.state.lucyAlgorithmFreeze = screenAnalyzer.FreezeDuration
					lucyAlgorithmCompaction(c)
				}
				break
			}
			c.attack()
			c.sleep(100 * time.Millisecond)
		}
		c.requestSwitch()
		return
	}

	c.attackFor(800 * time.Millisecond)
	c.requestSwitch()
}

func lucyClickResonance(c combatActor) bool {
	start := time.Now()
	clicked := false
	for c.resonanceChainAvailable() && time.Since(start) < 800*time.Millisecond {
		if c.currentResonance() > 0 && c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

func lucyClickLiberation(c combatActor) bool {
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

// lucyAlgorithmCompaction mirrors ok-ww Lucy.perform_algorithm_compaction():
// 5s loop: forte_full→multi_threading→liberation / liberation → click cycle.
func lucyAlgorithmCompaction(c combatActor) bool {
	if c.state.lucyAlgorithmUntil.IsZero() {
		return false
	}
	defer func() {
		c.state.lucyAlgorithmUntil = time.Time{}
		c.state.lucyAlgorithmFreeze = 0
	}()
	start := c.state.lucyAlgorithmUntil
	startFreeze := c.state.lucyAlgorithmFreeze
	for c.freezeElapsed(start, startFreeze) < 5*time.Second {
		if c.needFastPerform() {
			return false
		}
		if c.forteFull() && lucyPerformMultiThreading(c) {
			if lucyClickLiberation(c) {
				c.fBreak()
			}
			return true
		}
		if lucyClickLiberation(c) {
			c.fBreak()
			return true
		}
		cycleStart := time.Now()
		cycleFreeze := screenAnalyzer.FreezeDuration
		c.attack()
		if wait := 100*time.Millisecond - c.freezeElapsed(cycleStart, cycleFreeze); wait > 0 {
			c.sleep(wait)
		}
	}
	return false
}

// lucyPerformMultiThreading mirrors ok-ww Lucy.perform_multi_threading():
// heavy_click_forte→normal_attack(0.45s).
func lucyPerformMultiThreading(c combatActor) bool {
	if !c.forteFull() {
		return false
	}
	c.holdHeavyUntil(2*time.Second, 100*time.Millisecond, func() bool {
		return !c.forteFull()
	})
	if c.forteFull() {
		return false
	}
	c.state.lastHeavy = time.Now()
	c.attackFor(450 * time.Millisecond)
	return true
}

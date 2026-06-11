package combat

import "time"

type zaniWaitState int

const (
	zaniWaitFailed zaniWaitState = iota
	zaniWaitDone
	zaniWaitFull
	zaniWaitInterrupted
)

// performZani mirrors ok-ww Zani.do_perform():
//
//	intro(1.3s attack) → wait_down → check_liber →
//	in_liberation: should_end→liber2 / nightfall_combo → switch /
//	!in_liberation: f_break→echo→crisis_protocol→basic_attack_breakthrough→
//	  prepared→liberation→nightfall_combo → crisis_response_protocol → switch
func performZani(c combatActor) {
	zaniSyncLiberationState(c)
	zaniResetExpiredWindow(c)

	if c.state.zaniInLiberation && zaniLiberationTimeLeft(c) <= 0 {
		zaniEndLiberation(c)
	}

	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		c.attackFor(1300 * time.Millisecond)
	} else {
		c.sleep(10 * time.Millisecond)
	}
	c.waitDown(1200 * time.Millisecond)

	if c.state.zaniInLiberation {
		if zaniShouldEndLiberation(c, false) {
			zaniClickLiberation2(c)
		} else {
			zaniNightfallCombo(c, false)
		}
		c.requestSwitch()
		return
	}

	c.fBreak()
	if c.currentEcho() > 0.05 {
		c.echoImmediate()
	}

	castLiberation := false
	waitCrisisBeforeLiberation := false
	if !c.state.lastCrisis.IsZero() && c.freezeElapsed(c.state.lastCrisis, c.state.zaniCrisisFreeze) < 2450*time.Millisecond {
		zaniWaitCrisisProtocolEnd(c)
		if zaniCrisisTimeLeft(c) > -1*time.Second && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) && c.zaniPrepared() {
			castLiberation = true
			waitCrisisBeforeLiberation = screenAnalyzer.ZaniBlazesPct < 0.99
		} else {
			c.attackFor(250 * time.Millisecond)
			c.sleep(250 * time.Millisecond)
		}
		c.state.lastCrisis = time.Time{}
		c.state.zaniCrisisFreeze = 0
	}

	if !castLiberation {
		c.state.zaniLastChair = time.Time{}
		if !c.recentlySwitchedIn(1800*time.Millisecond) &&
			!c.isFirstEngage() &&
			!c.state.zaniLastLiberation2.IsZero() &&
			c.freezeElapsed(c.state.zaniLastLiberation2, c.state.zaniLiberation2Freeze) >= 2600*time.Millisecond {
			if !c.state.zaniLastAttackBreakthrough.IsZero() &&
				c.freezeElapsed(c.state.zaniLastAttackBreakthrough, c.state.zaniAttackBreakFreeze) < 4*time.Second {
				c.rightClickFor(50 * time.Millisecond)
				c.state.zaniLastDodge = time.Now()
				c.state.zaniLastDodgeFreeze = screenAnalyzer.FreezeDuration
			} else {
				c.sleep(250 * time.Millisecond)
				c.attackFor(250 * time.Millisecond)
				c.state.zaniLastChair = time.Now()
				c.state.zaniLastChairFreeze = screenAnalyzer.FreezeDuration
			}
			c.state.zaniLastLiberation2 = time.Time{}
			c.state.zaniLiberation2Freeze = 0
			c.state.zaniLastAttackBreakthrough = time.Time{}
			c.state.zaniAttackBreakFreeze = 0
		}

		breakthroughResult := zaniBasicAttackBreakthroughCombo(c)
		if c.zaniPrepared() {
			libAvailable := screenAnalyzer.Liberation || c.currentLiberation() > 0.05
			if libAvailable {
				result := zaniWaitFailed
				if breakthroughResult == zaniWaitDone {
					result = zaniWaitForteFull(c, 2200*time.Millisecond, false, true)
					if result == zaniWaitDone {
						c.rightClickFor(50 * time.Millisecond)
						c.state.zaniLastDodge = time.Now()
						c.state.zaniLastDodgeFreeze = screenAnalyzer.FreezeDuration
					}
				}
				if breakthroughResult == zaniWaitInterrupted || result == zaniWaitInterrupted {
					c.waitDown(600 * time.Millisecond)
				}
				if zaniCrisisResponseProtocolCombo(c) {
					castLiberation = screenAnalyzer.Liberation || c.currentLiberation() > 0.05
				}
			} else if c.forteFull() && zaniCrisisResponseProtocolCombo(c) {
				castLiberation = screenAnalyzer.Liberation || c.currentLiberation() > 0.05
			}
			if castLiberation {
				waitCrisisBeforeLiberation = screenAnalyzer.ZaniBlazesPct < 0.99
			} else {
				c.requestSwitch()
				return
			}
		}
	}

	if castLiberation {
		if waitCrisisBeforeLiberation {
			zaniWaitCrisisProtocolEnd(c)
			c.state.lastCrisis = time.Time{}
			c.state.zaniCrisisFreeze = 0
		}
		if zaniClickLiberation1(c) {
			c.state.lastLiberation = time.Now()
			c.state.zaniLiberationFreeze = screenAnalyzer.FreezeDuration
			c.state.zaniInLiberation = true
			c.state.lastNightfall = time.Time{}
			c.state.zaniNightfallFreeze = 0
			c.rightClickFor(50 * time.Millisecond)
			c.state.zaniLastDodge = time.Now()
			c.state.zaniLastDodgeFreeze = screenAnalyzer.FreezeDuration
			c.attackFor(150 * time.Millisecond)
			zaniNightfallCombo(c, true)
			c.sleep(100 * time.Millisecond)
			if c.forteFull() {
				zaniNightfallCombo(c, false)
			}
			c.requestSwitch()
			return
		}
		c.requestSwitch()
		return
	}

	if c.forteFull() {
		zaniCrisisResponseProtocolCombo(c)
	}
	c.requestSwitch()
}

func zaniClickLiberation1(c combatActor) bool {
	if !c.param.UseLiberation {
		return false
	}
	start := time.Now()
	clicked := false
	for time.Since(start) < 800*time.Millisecond && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) {
		c.forceLiberation()
		clicked = true
		c.sleep(100 * time.Millisecond)
	}
	if !clicked {
		retryDeadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(retryDeadline) && c.currentLiberation() > 0.001 {
			c.forceLiberation()
			clicked = true
			c.sleep(100 * time.Millisecond)
		}
	}
	return confirmLiberationCast(c, clicked, 7*time.Second)
}

func zaniResetExpiredWindow(c combatActor) {
	zaniSyncLiberationState(c)
	if c.state.zaniInLiberation && zaniLiberationTimeLeft(c) <= 0 {
		zaniEndLiberation(c)
	}
}

func zaniLiberationTimeLeft(c combatActor) time.Duration {
	if c.state.lastLiberation.IsZero() {
		return 0
	}
	return 20*time.Second - c.freezeElapsed(c.state.lastLiberation, c.state.zaniLiberationFreeze)
}

func zaniCrisisTimeLeft(c combatActor) time.Duration {
	if c.state.lastCrisis.IsZero() {
		return 0
	}
	return 1600*time.Millisecond - c.freezeElapsed(c.state.lastCrisis, c.state.zaniCrisisFreeze)
}

func zaniNightfallTimeLeft(c combatActor) time.Duration {
	if c.state.lastNightfall.IsZero() {
		return 0
	}
	return 2200*time.Millisecond - c.freezeElapsed(c.state.lastNightfall, c.state.zaniNightfallFreeze)
}

func zaniShouldEndLiberation(c combatActor, timeOnly bool) bool {
	if zaniLiberationTimeLeft(c) < 1700*time.Millisecond {
		return true
	}
	if timeOnly || c.zaniNightfallReady() {
		return false
	}
	if zaniWaitResonanceNotGray(c, true, true, 2500*time.Millisecond) == zaniWaitInterrupted {
		return true
	}
	return !c.forteFull()
}

func zaniWaitCrisisProtocolEnd(c combatActor) {
	if zaniCrisisTimeLeft(c) <= 0 {
		return
	}
	if !c.state.lastResonance.IsZero() && c.freezeElapsed(c.state.lastResonance, c.state.zaniResonanceFreeze) < 5*time.Second {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) && zaniCrisisTimeLeft(c) > 0 {
			c.sleep(50 * time.Millisecond)
		}
		return
	}
	deadline := time.Now().Add(2500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if zaniCrisisTimeLeft(c) <= 0 || c.currentResonance() > 0.001 {
			return
		}
		c.sleep(50 * time.Millisecond)
	}
}

func zaniStandardDefenseProtocolCombo(c combatActor) zaniWaitState {
	if c.forteFull() {
		return zaniWaitFull
	}
	if c.currentResonance() <= 0.05 {
		return zaniWaitFailed
	}
	start := time.Now()
	clicked := false
	for c.currentResonance() > 0.05 && time.Since(start) < 200*time.Millisecond {
		if c.forceSkill() {
			clicked = true
		}
		c.sleep(50 * time.Millisecond)
	}
	if !clicked {
		return zaniWaitFailed
	}
	c.sleep(200 * time.Millisecond)
	c.attackFor(200 * time.Millisecond)
	return zaniWaitDone
}

func zaniBasicAttackBreakthroughCombo(c combatActor) zaniWaitState {
	if c.forteFull() {
		return zaniWaitFull
	}
	result := zaniBasicAttackBreakthrough(c)
	if result == zaniWaitDone {
		c.state.zaniLastAttackBreakthrough = time.Now()
		c.state.zaniAttackBreakFreeze = screenAnalyzer.FreezeDuration
	}
	return result
}

func zaniBasicAttackBreakthrough(c combatActor) zaniWaitState {
	waitChair := 1200 * time.Millisecond
	if c.state.zaniLastChair.IsZero() {
		result := zaniStandardDefenseProtocolCombo(c)
		if result == zaniWaitFailed {
			sleep := 300*time.Millisecond - c.freezeElapsed(c.state.zaniLastDodge, c.state.zaniLastDodgeFreeze)
			if sleep < 0 {
				sleep = 0
			}
			if result = zaniWaitForteFull(c, sleep, false, false); result != zaniWaitDone {
				return result
			}
			if result = zaniWaitForteFull(c, 600*time.Millisecond, true, false); result != zaniWaitDone {
				return result
			}
			waitChair = 1150 * time.Millisecond
			if result = zaniWaitForteFull(c, 850*time.Millisecond, true, false); result != zaniWaitDone {
				return result
			}
		} else if result == zaniWaitFull {
			return zaniWaitFull
		}
	} else {
		waitChair -= c.freezeElapsed(c.state.zaniLastChair, c.state.zaniLastChairFreeze)
		c.state.zaniLastChair = time.Time{}
		c.state.zaniLastChairFreeze = 0
	}
	if result := zaniWaitForteFull(c, waitChair, false, false); result != zaniWaitDone {
		return result
	}
	c.attackFor(200 * time.Millisecond)
	return zaniWaitDone
}

func zaniWaitForteFull(c combatActor, timeout time.Duration, sendAttack, checkForte bool) zaniWaitState {
	if timeout <= 0 {
		return zaniWaitDone
	}
	start := time.Now()
	lastCheck := time.Time{}
	lastValue := -1.0
	startedChecking := false
	for time.Since(start) < timeout {
		if c.forteFull() {
			return zaniWaitFull
		}
		if c.flying() {
			return zaniWaitInterrupted
		}
		if checkForte {
			if time.Since(start) > 800*time.Millisecond {
				startedChecking = true
			}
			if startedChecking && (lastCheck.IsZero() || time.Since(lastCheck) >= 200*time.Millisecond) {
				currentValue := c.zaniForteValue()
				if lastValue > 0 {
					gap := currentValue - lastValue
					if gap < 0.01 && !c.forteFull() {
						c.rightClickFor(50 * time.Millisecond)
						c.state.zaniLastDodge = time.Now()
						return zaniWaitInterrupted
					}
				}
				lastValue = currentValue
				lastCheck = time.Now()
			}
		}
		if sendAttack {
			c.attack()
		}
		c.sleep(100 * time.Millisecond)
	}
	return zaniWaitDone
}

func zaniWaitResonanceNotGray(c combatActor, sendClick, liberTimeCheck bool, timeout time.Duration) zaniWaitState {
	if timeout <= 0 {
		timeout = 2500 * time.Millisecond
	}
	start := time.Now()
	stableStart := time.Time{}
	for time.Since(start) < timeout {
		if c.currentResonance() > 0.001 {
			if stableStart.IsZero() {
				stableStart = time.Now()
			} else if time.Since(stableStart) >= 100*time.Millisecond {
				return zaniWaitDone
			}
		} else {
			stableStart = time.Time{}
		}
		if liberTimeCheck && zaniLiberationTimeLeft(c) < 1700*time.Millisecond {
			return zaniWaitInterrupted
		}
		if sendClick {
			c.attack()
		}
		c.sleep(50 * time.Millisecond)
	}
	return zaniWaitFailed
}

func zaniCrisisResponseProtocolCombo(c combatActor) bool {
	if !c.forteFull() {
		result := zaniWaitFailed
		if result = zaniBasicAttackBreakthrough(c); result == zaniWaitDone {
			result = zaniWaitForteFull(c, 2200*time.Millisecond, false, true)
		}
		if result != zaniWaitFull && !c.forteFull() {
			return false
		}
	}
	start := time.Now()
	clicked := false
	for time.Since(start) < time.Second {
		if !c.forteFull() {
			break
		}
		if c.forceSkill() {
			clicked = true
		}
		c.sleep(50 * time.Millisecond)
	}
	duration := time.Since(start)
	if !clicked || (c.forteFull() && duration < 350*time.Millisecond) {
		return false
	}
	c.state.lastCrisis = time.Now()
	c.state.zaniCrisisFreeze = screenAnalyzer.FreezeDuration
	c.state.zaniResonanceFreeze = screenAnalyzer.FreezeDuration
	return true
}

func zaniNightfallCombo(c combatActor, cancelLastSmash bool) {
	start := time.Now()
	if !c.zaniNightfallReady() {
		for (!c.zaniNightfallReady() || time.Since(start) < 1600*time.Millisecond) && time.Since(start) < 3500*time.Millisecond {
			if !c.state.zaniInLiberation || zaniLiberationTimeLeft(c) <= 0 {
				return
			}
			if zaniShouldEndLiberation(c, true) && zaniClickLiberation2(c) {
				return
			}
			c.attack()
			c.sleep(90 * time.Millisecond)
		}
	}
	c.attackFor(500 * time.Millisecond)
	if cancelLastSmash {
		cancelStart := time.Now()
		for c.zaniNightfallReady() && time.Since(cancelStart) < 2500*time.Millisecond {
			c.attack()
			c.sleep(90 * time.Millisecond)
		}
		c.sleep(250 * time.Millisecond)
		c.rightClickFor(100 * time.Millisecond)
		c.state.zaniLastDodge = time.Now()
		return
	}
	c.state.lastNightfall = time.Now()
	c.state.zaniNightfallFreeze = screenAnalyzer.FreezeDuration
}

func zaniClickLiberation2(c combatActor) bool {
	start := time.Now()
	sendKey := true
	for !screenAnalyzer.ZaniNotLiberBox {
		if time.Since(start) > 6*time.Second {
			zaniSyncLiberationState(c)
			if !c.state.zaniInLiberation {
				zaniEndLiberation(c)
			}
			return false
		}
		if c.currentResonance() <= 0.001 {
			start = time.Now()
		} else if time.Since(start) > 1500*time.Millisecond {
			sendKey = false
		}
		if sendKey {
			c.forceLiberation()
		}
		c.sleep(50 * time.Millisecond)
		zaniSyncLiberationState(c)
	}
	if time.Since(start) >= 2250*time.Millisecond {
		c.state.zaniLastLiberation2 = time.Now()
		c.state.zaniLiberation2Freeze = screenAnalyzer.FreezeDuration
		c.addFreezeDuration(2250 * time.Millisecond)
	}
	zaniEndLiberation(c)
	return true
}

func zaniEndLiberation(c combatActor) {
	c.state.zaniInLiberation = false
	c.state.lastCrisis = time.Time{}
	c.state.zaniCrisisFreeze = 0
	c.state.lastNightfall = time.Time{}
	c.state.zaniNightfallFreeze = 0
	c.state.lastLiberation = time.Time{}
	c.state.zaniLiberationFreeze = 0
}

func zaniSyncLiberationState(c combatActor) {
	if screenAnalyzer.ZaniNotLiberBox {
		c.state.zaniInLiberation = false
		return
	}
	if screenAnalyzer.ZaniLiberBox {
		c.state.zaniInLiberation = true
	}
}

package combat

import "time"

// performZhezhi mirrors ok-ww Zhezhi.do_perform():
//
//	carlotta_in_team → do_perform_interlock / normal:
//	  intro(1.5s attack) → liberation → resonance_blue(resonance_until_not_blue) → switch /
//	  resonance+forte_full→resonance+normal_attack(set blue) → echo → switch
func performZhezhi(c combatActor) {
	if c.teamHasAny("carlotta", "carlotta2") {
		performZhezhiInterlock(c)
		return
	}
	if c.recentlyIntroSwitchedIn(1700 * time.Millisecond) {
		c.attackFor(1500 * time.Millisecond)
	}
	zhezhiClickLiberation(c)
	if (c.state.zhezhiBlueReady || c.zhezhiBlueReady()) && c.currentResonance() > 0.05 {
		c.state.zhezhiBlueReady = false
		zhezhiResonanceUntilNotBlue(c, false)
		c.requestSwitch()
		return
	}
	if c.currentResonance() > 0.05 && c.forteFull() && zhezhiClickResonance(c) {
		c.attackFor(800 * time.Millisecond)
		c.state.zhezhiBlueReady = true
		c.requestSwitch()
		return
	}
	if !zhezhiEcho(c) {
		c.attackFor(100 * time.Millisecond)
	}
	c.requestSwitch()
}

// performZhezhiInterlock mirrors ok-ww Zhezhi.do_perform_interlock():
// carlotta interlock: intro(1.3s attack)→flying→wait_down→right_click →
//   !blue+forte<3: normal_attack→switch /
//   !blue+resonance+forte>1: liberation+resonance+normal_attack /
//   con_lock: resonance_until_not_blue→liberation/echo → echo→switch
func performZhezhiInterlock(c combatActor) {
	if c.recentlyIntroSwitchedIn(1500 * time.Millisecond) {
		c.attackFor(1300 * time.Millisecond)
	}
	if c.flying() {
		c.waitDown(1200 * time.Millisecond)
		c.rightClick()
		c.sleep(50 * time.Millisecond)
	}
	forteTier := c.zhezhiForteTier()
	carlottaReady := zhezhiCarlottaReady(c)
	conFullAndCarlottaReady := screenAnalyzer.ConcertoPct >= 1.0 && carlottaReady
	if !c.zhezhiBlueReady() && forteTier < 3 && !conFullAndCarlottaReady {
		c.attackFor(1400 * time.Millisecond)
		if !carlottaReady {
			c.requestSwitch()
			return
		}
	}
	if !c.zhezhiBlueReady() && c.currentResonance() > 0.05 && forteTier > 1 && !conFullAndCarlottaReady {
		if zhezhiConLock(c) {
			if zhezhiClickLiberationWait(c, 500*time.Millisecond) {
				c.sleep(200 * time.Millisecond)
			}
		}
		if zhezhiClickResonance(c) {
			c.attackFor(800 * time.Millisecond)
		}
	}
	if zhezhiConLock(c) {
		if c.zhezhiBlueReady() && c.currentResonance() > 0.05 {
			zhezhiResonanceUntilNotBlue(c, true)
			if zhezhiConLock(c) {
				if zhezhiClickLiberationWait(c, 500*time.Millisecond) {
					c.sleep(200 * time.Millisecond)
				}
			} else if zhezhiEcho(c) {
				c.rightClick()
				c.sleep(50 * time.Millisecond)
			}
		}
	}
	zhezhiEchoWait(c, 2*time.Second)
	c.requestSwitch()
}

func zhezhiClickLiberation(c combatActor) bool {
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
	return finishLiberationCast(c, clicked, 7*time.Second)
}

func zhezhiClickLiberationWait(c combatActor, wait time.Duration) bool {
	if wait > 0 {
		deadline := time.Now().Add(wait)
		for time.Now().Before(deadline) {
			if zhezhiClickLiberation(c) {
				return true
			}
			if c.currentLiberation() > 0.001 {
				c.sleep(100 * time.Millisecond)
				continue
			}
			break
		}
	}
	return zhezhiClickLiberation(c)
}

func zhezhiClickResonance(c combatActor) bool {
	start := time.Now()
	clicked := false
	for c.currentResonance() > 0.05 && time.Since(start) < 15*time.Second {
		if c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

// zhezhiResonanceUntilNotBlue mirrors ok-ww Zhezhi.resonance_until_not_blue():
// holds mouse_down while resonance_available+blue, periodic jump, up to 4s.
func zhezhiResonanceUntilNotBlue(c combatActor, stopOnConcerto bool) {
	start := time.Now()
	waitBlueDeadline := time.Now().Add(300 * time.Millisecond)
	for !c.zhezhiBlueReady() && time.Now().Before(waitBlueDeadline) {
		c.sleep(50 * time.Millisecond)
	}
	nextJump := time.Now().Add(400 * time.Millisecond)
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostTouchDown(0, 640, 360, 1).Wait()
	defer ctrl.PostTouchUp(0).Wait()
	for c.currentResonance() > 0.05 && c.zhezhiBlueReady() {
		c.forceSkill()
		if c.needFastPerform() && time.Since(start) > 1100*time.Millisecond {
			break
		}
		if screenAnalyzer.ConcertoPct >= 1.0 {
			if !c.teamHasAny("carlotta", "carlotta2") || (stopOnConcerto && zhezhiConLock(c)) {
				break
			}
		}
		if time.Since(start) > 4*time.Second {
			break
		}
		if time.Now().After(nextJump) {
			c.jumpAttackFor(50 * time.Millisecond)
			nextJump = time.Now().Add(400 * time.Millisecond)
		}
		c.sleep(90 * time.Millisecond)
	}
}

func zhezhiCarlottaReady(c combatActor) bool {
	carlottaState := c.action.charStates["carlotta"]
	if carlottaState == nil {
		carlottaState = c.action.charStates["carlotta2"]
	}
	if carlottaState == nil {
		return false
	}
	return carlottaReady(combatActor{action: c.action, ctx: c.ctx, param: c.param, state: carlottaState})
}

func zhezhiConLock(c combatActor) bool {
	return zhezhiCarlottaReady(c) || screenAnalyzer.ConcertoPct < 0.6 || (screenAnalyzer.ConcertoPct >= 1.0 && !zhezhiCarlottaReady(c))
}

func zhezhiEcho(c combatActor) bool {
	if c.currentEcho() <= 0.05 {
		return false
	}
	c.run("Combat_RotationEcho")
	c.state.lastEcho = time.Now()
	return true
}

func zhezhiEchoWait(c combatActor, wait time.Duration) bool {
	if wait <= 0 {
		return zhezhiEcho(c)
	}
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		if zhezhiEcho(c) {
			return true
		}
		c.sleep(100 * time.Millisecond)
	}
	return zhezhiEcho(c)
}

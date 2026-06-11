package combat

import "time"

// performLuhesi mirrors ok-ww Luhesi.do_perform():
//
//	normal_attack(1.1s) → !sub_dps_intro→resonance→switch /
//	sub_dps_intro: jump→flying → 12s loop:
//	  elbow_strike(right_click) / handle_heavy→liberation / try_jump→kick→liberation / check_res→jump / click
func performLuhesi(c combatActor) {
	c.attackFor(1100 * time.Millisecond)
	if !(c.recentlyIntroSwitchedIn(2*time.Second) && c.switchedFromRole(roleSub, 2*time.Second)) {
		c.forceSkill()
		c.sleep(100 * time.Millisecond)
		c.requestSwitch()
		return
	}
	c.state.lastIntro = time.Now()
	if !c.flying() {
		waitDeadline := time.Now().Add(200 * time.Millisecond)
		for !c.flying() && time.Now().Before(waitDeadline) {
			c.jump()
			c.sleep(100 * time.Millisecond)
		}
	}
	resCount := 0
	tryJump := false
	detectReady := c.currentEcho() > 0
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		if luhesiDetectElbowStrike(c, detectReady) {
			elbowDeadline := time.Now().Add(1500 * time.Millisecond)
			for time.Now().Before(elbowDeadline) && luhesiDetectElbowStrike(c, detectReady) {
				c.rightClickFor(50 * time.Millisecond)
			}
		} else if luhesiHandleHeavy(c, resCount) && resCount > 2 {
			luhesiLiberate(c)
			c.requestSwitch()
			return
		} else if tryJump && !luhesiCheckRes(c) {
			waitKickDeadline := time.Now().Add(1 * time.Second)
			for time.Now().Before(waitKickDeadline) && !c.luhesiKickReady() && !luhesiDetectElbowStrike(c, detectReady) {
				c.forceSkill()
				c.sleep(100 * time.Millisecond)
			}
			if luhesiDetectElbowStrike(c, detectReady) {
				continue
			}
			tryJump = false
			if c.luhesiKickReady() {
				resCount++
			} else {
				if resCount == 2 {
					c.waitDown(1200 * time.Millisecond)
				}
				luhesiLiberate(c)
				c.requestSwitch()
				return
			}
		} else if luhesiCheckRes(c) {
			c.jump()
			tryJump = true
		} else {
			c.attack()
		}
		c.sleep(10 * time.Millisecond)
	}
	c.requestSwitch()
}

func luhesiDetectElbowStrike(c combatActor, detectReady bool) bool {
	return detectReady && c.currentEcho() == 0
}

func luhesiCheckRes(c combatActor) bool {
	return c.hasLongAction2()
}

// luhesiHandleHeavy mirrors ok-ww Luhesi.handle_heavy():
// loop while luhesi_kick icon visible (3s): f_break + attack.
func luhesiHandleHeavy(c combatActor, resCount int) bool {
	kickStart := time.Now()
	haveKick := false
	for c.luhesiKickReady() && time.Since(kickStart) < 3*time.Second {
		haveKick = true
		if resCount < 3 {
			c.fBreak()
		}
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	return haveKick
}

// luhesiLiberate mirrors ok-ww Luhesi.lib():
// casts liberation when luhesi_lib available, with f_break on success.
func luhesiLiberate(c combatActor) bool {
	if !c.luhesiLibReady() {
		return false
	}
	start := time.Now()
	clicked := false
	for c.luhesiLibReady() && time.Since(start) < 800*time.Millisecond {
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
	if !clicked || c.luhesiLibReady() {
		return false
	}
	if !finishLiberationCast(c, true, 7*time.Second) {
		return false
	}
	c.fBreak()
	return true
}

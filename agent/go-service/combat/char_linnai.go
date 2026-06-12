package combat

import "time"

// performLinnai mirrors ok-ww Linnai.do_perform():
//
//	intro: check_res(has_long_action2)→attack(1.33s) / else→attack(1s)+echo+liberation+resonance+
//	  wait_mouse_forte_full→hold_attack→perform_under_intro → switch
//	!intro: echo→perform_under_intro / flying→attack / liberation→attack / resonance→switch
func performLinnai(c combatActor) {
	if c.recentlyIntroSwitchedIn(1700 * time.Millisecond) {
		if c.hasLongAction2() {
			c.attackFor(1330 * time.Millisecond)
		} else {
			c.attackFor(1 * time.Second)
			c.echoImmediate()
			if screenAnalyzer.ConcertoPct < 1.0 {
				linnaiClickLiberation(c)
			}
			if !c.mouseForteFull() {
				linnaiClickResonance(c, 1200*time.Millisecond)
			}
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) && !c.mouseForteFull() {
				c.attack()
				c.sleep(100 * time.Millisecond)
			}
			if c.mouseForteFull() {
				if linnaiHoldAttackUntil(c, 5*time.Second, 100*time.Millisecond, func() bool {
					return !c.mouseForteFull()
				}) {
					c.sleep(400 * time.Millisecond)
					linnaiPerformUnderIntro(c)
				}
			}
			c.requestSwitch()
			return
		}
	} else {
		c.echoImmediate()
		if linnaiPerformUnderIntro(c) {
		} else if c.flying() {
			c.attackFor(100 * time.Millisecond)
		} else if screenAnalyzer.ConcertoPct < 1.0 && linnaiClickLiberation(c) {
			c.attackFor(500 * time.Millisecond)
		}
		linnaiClickResonance(c, 15*time.Second)
		c.requestSwitch()
		return
	}

	c.requestSwitch()
}

func linnaiClickLiberation(c combatActor) bool {
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

func linnaiClickResonance(c combatActor, timeout time.Duration) bool {
	start := time.Now()
	clicked := false
	for c.resonanceChainAvailable() && time.Since(start) < timeout {
		if c.currentResonance() > 0 && c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

func linnaiHoldAttackUntil(c combatActor, maxDuration, poll time.Duration, stop func() bool) bool {
	if maxDuration <= 0 {
		return false
	}
	if poll <= 0 {
		poll = 100 * time.Millisecond
	}
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostTouchDown(0, 640, 360, 1).Wait()
	defer ctrl.PostTouchUp(0).Wait()
	deadline := time.Now().Add(maxDuration)
	for time.Now().Before(deadline) {
		if stop != nil && stop() {
			return true
		}
		c.sleep(poll)
	}
	return false
}

// linnaiPerformUnderIntro mirrors ok-ww Linnai.perform_under_intro():
// check_res→wait_color_full→drain_forte(jump)→resonance_kick→resonance_kick2→liberation.
func linnaiPerformUnderIntro(c combatActor) bool {
	if !c.hasLongAction2() {
		return false
	}

	waitColorDeadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(waitColorDeadline) && !c.linnaiColorFull() && screenAnalyzer.ConcertoPct < 1.0 {
		c.attack()
		c.sleep(100 * time.Millisecond)
	}

	drainSucceeded := false
	drainDeadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(drainDeadline) && c.forteFull() {
		c.jump()
		c.sleep(100 * time.Millisecond)
		if !c.forteFull() {
			drainSucceeded = true
			break
		}
	}

	if drainSucceeded {
		firstKick := linnaiWaitResonanceKick(c, 2*time.Second)
		if firstKick {
			linnaiWaitAfterResonanceKick(c)
		}

		secondKick := false
		secondKickDeadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(secondKickDeadline) {
			if screenAnalyzer.ConcertoPct >= 1.0 {
				break
			}
			if linnaiClickResonance(c, 300*time.Millisecond) {
				secondKick = true
				break
			}
			c.attack()
			c.sleep(100 * time.Millisecond)
		}
		if secondKick {
			linnaiWaitAfterResonanceKick(c)
		}
	}

	if screenAnalyzer.ConcertoPct < 1.0 && linnaiClickLiberation(c) {
		libDeadline := time.Now().Add(1200 * time.Millisecond)
		for time.Now().Before(libDeadline) && screenAnalyzer.ConcertoPct < 1.0 {
			c.attack()
			c.sleep(90 * time.Millisecond)
		}
	}

	return true
}

func linnaiWaitResonanceKick(c combatActor, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if linnaiClickResonance(c, 300*time.Millisecond) {
			return true
		}
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	return false
}

func linnaiWaitAfterResonanceKick(c combatActor) {
	c.sleep(300 * time.Millisecond)
	c.waitDown(1200 * time.Millisecond)
}

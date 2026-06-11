package combat

import "time"

// performCamellya mirrors ok-ww Camellya.do_perform():
//
//	intro(1.2s attack + 4.6s heavy_attack until_con_full) → liberation(con<0.82) →
//	con<0.82: resonance + 1.1s-3.1s attack loop → con>=0.82: 4.6s attack loop →
//	budding state: ephemeral_cast → held heavy + liberation at 1.5s →
//	finish: mouse_up + budding_resonance + echo → switch
func performCamellya(c combatActor) {
	if c.recentlyIntroSwitchedIn(1600 * time.Millisecond) {
		c.attackFor(1200 * time.Millisecond)
		c.sleep(100 * time.Millisecond)
		camellyaHeavyAttack(c, 4600*time.Millisecond, true, false)
		c.state.lastHeavy = time.Now()
	}

	if screenAnalyzer.ConcertoPct < 0.82 && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) {
		if camellyyaClickLiberation(c) {
			c.attackFor(300 * time.Millisecond)
		}
	}

	loopTime := 1100 * time.Millisecond
	if screenAnalyzer.ConcertoPct >= 0.82 {
		loopTime = 4600 * time.Millisecond
	} else if c.currentResonance() > 0.05 {
		camellyyaClickResonance(c)
	}

	start := time.Now()
	buddingStart := start
	budding := false
	heavyAtt := false
	freezeForteCheck := false
	freezeForteUntil := time.Time{}
	camellyyaResetHeavyRetry(c)

	for time.Since(buddingStart) < loopTime || c.camellyaBuddingActive() {
		if !budding {
			if c.camellyaEphemeralReady() && screenAnalyzer.ConcertoPct >= 1.0 {
				camellyaEphemeralCast(c)
				budding = true
			} else {
				c.attack()
				c.sleep(100 * time.Millisecond)
				if screenAnalyzer.ConcertoPct < 0.82 {
					if screenAnalyzer.ConcertoPct < 1.0 {
						camellyyaEcho(c)
						c.requestSwitch()
						return
					}
					if loopTime < 3100*time.Millisecond {
						loopTime += 1 * time.Second
					}
				}
			}
			if budding {
				buddingStart = time.Now()
				loopTime = 5100 * time.Millisecond
				c.state.camellyaInBudding = true
				camellyaCheckTarget(c, false)
			}
		}

		if budding {
			if !heavyAtt {
				heavyAtt = true
				c.ctx.GetTasker().GetController().PostTouchDown(0, 640, 360, 1).Wait()
			}
			if time.Since(buddingStart) < 1500*time.Millisecond && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) {
				if camellyyaClickLiberation(c) {
					c.ctx.GetTasker().GetController().PostTouchUp(0).Wait()
					c.sleep(200 * time.Millisecond)
					c.ctx.GetTasker().GetController().PostTouchDown(0, 640, 360, 1).Wait()
				}
			}
			camellyaCheckTarget(c, true)
			if !freezeForteCheck {
				if camellyaShouldRetryHeavyAttack(c, true) < 0 {
					freezeForteCheck = true
					freezeForteUntil = time.Now().Add(200 * time.Millisecond)
				}
			} else if time.Now().After(freezeForteUntil) {
				freezeForteCheck = false
			}
		}

		camellyaCheckTarget(c, heavyAtt)
		c.sleep(100 * time.Millisecond)
	}

	if heavyAtt {
		c.ctx.GetTasker().GetController().PostTouchUp(0).Wait()
		c.sleep(100 * time.Millisecond)
	}
	if budding {
		c.state.camellyaInBudding = false
		camellyyaClickResonance(c)
		c.sleep(100 * time.Millisecond)
	}
	camellyyaEcho(c)
	c.requestSwitch()
}

// camellyyaEcho mirrors ok-ww Camellya.click_echo():
// sends echo key immediately if available.
func camellyyaEcho(c combatActor) bool {
	if c.currentEcho() <= 0.05 {
		return false
	}
	c.run("Combat_RotationEcho")
	c.state.lastEcho = time.Now()
	return true
}

// camellyyaClickResonance mirrors ok-ww Camellya.click_resonance():
// casts resonance while available for up to 15s.
func camellyyaClickResonance(c combatActor) bool {
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

// camellyyaClickLiberation mirrors ok-ww Camellya.click_liberation():
// casts liberation with standard retry; finishLiberationCast on success.
func camellyyaClickLiberation(c combatActor) bool {
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

// camellyaEphemeralCast mirrors ok-ww Camellya.ephemeral_cast():
// spams resonance while ephemeral_ready, then sleeps 1.1s.
func camellyaEphemeralCast(c combatActor) {
	for c.camellyaEphemeralReady() {
		c.forceSkill()
		c.sleep(100 * time.Millisecond)
	}
	c.sleep(1100 * time.Millisecond)
}

// camellyaHeavyAttack mirrors ok-ww Camellya.heavy_attack():
// held heavy for duration with forte-drop retry and target-check logic.
func camellyaHeavyAttack(c combatActor, duration time.Duration, untilConFull bool, budding bool) {
	ctrl := c.ctx.GetTasker().GetController()
	freezeForteCheck := false
	freezeForteUntil := time.Time{}
	camellyyaResetHeavyRetry(c)
	ctrl.PostTouchDown(0, 640, 360, 1).Wait()
	start := time.Now()
	for time.Since(start) < duration {
		if untilConFull && screenAnalyzer.ConcertoPct >= 1.0 {
			break
		}
		if 0 <= c.camellyaForteValue(budding) && c.camellyaForteValue(budding) <= 0.01 {
			break
		}
		camellyaCheckTarget(c, true)
		if !freezeForteCheck {
			if camellyaShouldRetryHeavyAttack(c, budding) < 0 {
				freezeForteCheck = true
				freezeForteUntil = time.Now().Add(200 * time.Millisecond)
			}
		} else if time.Now().After(freezeForteUntil) {
			freezeForteCheck = false
		}
		c.sleep(100 * time.Millisecond)
	}
	c.sleep(100 * time.Millisecond)
	ctrl.PostTouchUp(0).Wait()
	camellyyaResetHeavyRetry(c)
}

// camellyaShouldRetryHeavyAttack mirrors ok-ww Camellya.should_retry_heavy_attack():
// monitors forte drop rate; releases+re-presses heavy if forte stalls >0.6s.
func camellyaShouldRetryHeavyAttack(c combatActor, budding bool) int {
	currentForte := c.camellyaForteValue(budding)
	if currentForte < 0 {
		return -1
	}
	diff := c.state.lastForte - currentForte
	if !c.state.waitingForForteDrop && 0 <= diff && diff <= 0.01 {
		c.state.waitingForForteDrop = true
		c.state.forteDropAt = time.Now()
		c.state.forteDropFreeze = screenAnalyzer.FreezeDuration
		c.state.forteDropStartForte = currentForte
	}
	if c.state.waitingForForteDrop && c.state.forteDropStartForte-currentForte > 0.01 {
		c.state.waitingForForteDrop = false
	}
	if c.state.waitingForForteDrop && !c.state.forteDropAt.IsZero() && c.freezeElapsed(c.state.forteDropAt, c.state.forteDropFreeze) > 600*time.Millisecond {
		ctrl := c.ctx.GetTasker().GetController()
		ctrl.PostTouchUp(0).Wait()
		c.sleep(100 * time.Millisecond)
		ctrl.PostTouchDown(0, 640, 360, 1).Wait()
		c.sleep(100 * time.Millisecond)
		camellyyaResetHeavyRetry(c)
		c.state.lastForte = currentForte
		return -1
	}
	if !c.state.waitingForForteDrop {
		c.state.forteDropAt = time.Time{}
		c.state.forteDropFreeze = 0
	}
	c.state.lastForte = currentForte
	return 0
}

func camellyyaResetHeavyRetry(c combatActor) {
	c.state.lastForte = 0
	c.state.waitingForForteDrop = false
	c.state.forteDropAt = time.Time{}
	c.state.forteDropFreeze = 0
	c.state.forteDropStartForte = 0
}

// camellyaCheckTarget mirrors ok-ww Camellya.check_target():
// releases heavy if target lost, re-checks combat and resumes.
func camellyaCheckTarget(c combatActor, heavy bool) {
	if screenAnalyzer.HasTarget {
		return
	}
	ctrl := c.ctx.GetTasker().GetController()
	if heavy {
		ctrl.PostTouchUp(0).Wait()
	}
	c.attack()
	c.sleep(100 * time.Millisecond)
	if heavy {
		ctrl.PostTouchDown(0, 640, 360, 1).Wait()
	}
}

package combat

import "time"

// performCartethyia mirrors ok-ww Cartethyia.do_perform():
//
//	dual-mode: is_small(Cartethyia) vs fleurdelys(transformed).
//	Common flow: intro/echo → if is_small: wait_down + acquire_buffs + mid_air + liberation(transform) →
//	  click_resonance_with_big_lib → attack loop → try_lib_big → switch
func performCartethyia(c combatActor) {
	// Python: self.transform = False — reset per call
	c.state.transformed = false

	if c.recentlyIntroSwitchedIn(1700 * time.Millisecond) {
		c.attackFor(1200 * time.Millisecond)
	} else {
		c.echo()
	}

	if c.cartethyiaIsSmall() {
		if c.flying() {
			c.waitDown(1200 * time.Millisecond)
		}
		if cartethyiaAcquireMissingBuffs(c) {
			c.requestSwitch()
			return
		}
		// Python: check_combat between buffs and mid-air
		cartethyiaTryMidAirAttack(c)
		// Python: check_combat after mid-air
		if cartethyiaClickLiberation(c) {
			c.state.transformed = true
		} else if !c.cartethyiaIsSmall() {
			c.state.transformed = true
		}
	}

	// Python: common path for BOTH forms — click_resonance_with_lib_big
	if cartethyiaClickResonanceWithBigLiber(c) {
		// success: nothing else needed
	} else {
		window := 1100 * time.Millisecond
		if !c.cartethyiaIsSmall() {
			window = cartethyiaTransformAttackWindow(c)
		}
		start := time.Now()
		for time.Since(start) < window {
			if cartethyiaTryBigLiber(c) {
				c.requestSwitch()
				return
			}
			c.attack()
			c.sleep(100 * time.Millisecond)
		}
		if !c.cartethyiaIsSmall() {
			c.state.cartethyiaN4At = time.Now()
			c.state.cartethyiaN4Freeze = screenAnalyzer.FreezeDuration
		}
	}
	cartethyiaTryBigLiber(c)
	c.requestSwitch()
}

// cartethyiaAcquireMissingBuffs mirrors ok-ww Cartethyia.acquire_missing_buffs():
// acquires sword buffs: sword2(click loop) → sword3(resonance) → sword1(heavy) → sets try_mid_air_attack flag.
func cartethyiaAcquireMissingBuffs(c combatActor) bool {
	sword1, sword2, sword3 := c.cartethyiaSwordBuffs()
	if sword1 && sword2 && sword3 {
		c.state.cartethyiaTryMidAirOnce = false
		return false
	}

	hasPerformAction := !(sword2 && sword3)
	if !sword2 {
		timeout := 3500 * time.Millisecond
		start := time.Now()
		interruptHandled := false
		for time.Since(start) < timeout {
			_, sword2, _ = c.cartethyiaSwordBuffs()
			if sword2 {
				break
			}
			if !interruptHandled && c.flying() {
				interruptHandled = true
				c.waitDown(3 * time.Second)
				start = time.Now()
			}
			c.attack()
			c.sleep(100 * time.Millisecond)
		}
	}

	resUsed := false
	_, _, sword3 = c.cartethyiaSwordBuffs()
	if !sword3 {
		resUsed = cartethyiaClickResonance(c, 1200*time.Millisecond)
	}
	if c.liberationAvailable() {
		if resUsed {
			c.sleep(200 * time.Millisecond)
		}
	} else if hasPerformAction {
		return true
	}

	sword1, _, _ = c.cartethyiaSwordBuffs()
	if !sword1 {
		ctrl := c.ctx.GetTasker().GetController()
		ctrl.PostTouchDown(0, 640, 360, 1).Wait()
		start := time.Now()
		for time.Since(start) < 1500*time.Millisecond {
			sword1, _, _ = c.cartethyiaSwordBuffs()
			if sword1 {
				break
			}
			c.sleep(50 * time.Millisecond)
		}
		ctrl.PostTouchUp(0).Wait()
	}

	sword1, sword2, sword3 = c.cartethyiaSwordBuffs()
	c.state.cartethyiaTryMidAirOnce = !sword1 && !sword2 && !sword3

	return !c.liberationAvailable()
}

// cartethyiaTransformAttackWindow mirrors ok-ww Cartethyia.fleurdelys_n4_duration():
// returns N4 attack window duration for transformed state based on intro/transform/backswing state.
func cartethyiaTransformAttackWindow(c combatActor) time.Duration {
	defer func() {
		c.state.cartethyiaN4At = time.Time{}
		c.state.cartethyiaN4Freeze = 0
		c.state.cartethyiaResAt = time.Time{}
		c.state.cartethyiaResFreeze = 0
	}()

	if !c.state.transformed && c.recentlyIntroSwitchedIn(1700*time.Millisecond) {
		window := 3900*time.Millisecond - c.performElapsed()
		if window > 1100*time.Millisecond {
			return window
		}
	}
	if c.state.transformed || c.isFirstEngage() || c.freezeElapsed(c.state.cartethyiaN4At, c.state.cartethyiaN4Freeze) < 1500*time.Millisecond {
		return 3250 * time.Millisecond
	}
	if c.state.cartethyiaN4At.IsZero() {
		return 0
	}
	if !c.state.cartethyiaResAt.IsZero() {
		if backswing := c.freezeElapsed(c.state.cartethyiaResAt, c.state.cartethyiaResFreeze); backswing < 2500*time.Millisecond {
			return 2*time.Second + maxDuration(0, 1600*time.Millisecond-backswing)
		}
	}
	if !c.state.transformUntil.IsZero() {
		window := time.Until(c.state.transformUntil)
		if window > 0 {
			if window < 1200*time.Millisecond {
				return 1200 * time.Millisecond
			}
			return window
		}
	}
	window := 1900*time.Millisecond - c.performElapsed()
	if window < 1100*time.Millisecond {
		return 1100 * time.Millisecond
	}
	return window
}

// cartethyiaClickResonanceWithBigLiber mirrors ok-ww Cartethyia.click_resonance_with_lib_big():
// casts resonance with big liberation check; handles resonance<0.17 attack padding.
func cartethyiaClickResonanceWithBigLiber(c combatActor) bool {
	if !c.resonanceAvailable() {
		return false
	}
	start := time.Now()
	clicked := false
	resonanceStarted := time.Time{}
	for c.resonanceAvailable() && time.Since(start) < 8*time.Second {
		if cartethyiaTryBigLiber(c) {
			return true
		}
		if c.currentResonance() < 0.17 && !resonanceStarted.IsZero() && time.Since(resonanceStarted) < 2500*time.Millisecond {
			c.attack()
			c.sleep(100 * time.Millisecond)
			continue
		}
		if c.currentResonance() > 0 && c.forceSkill() {
			clicked = true
			if resonanceStarted.IsZero() {
				resonanceStarted = time.Now()
				c.state.cartethyiaResAt = resonanceStarted
				c.state.cartethyiaResFreeze = screenAnalyzer.FreezeDuration
			}
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked && cartethyiaTryBigLiber(c)
}

// cartethyiaTryBigLiber mirrors ok-ww Cartethyia.try_lib_big():
// casts liberation + resonance if big liberation icon is visible.
func cartethyiaTryBigLiber(c combatActor) bool {
	if !c.cartethyiaBigLiberAvailable() {
		return false
	}
	if !cartethyiaClickLiberation(c) {
		return false
	}
	cartethyiaClickResonance(c, 800*time.Millisecond)
	return true
}

// cartethyiaClickLiberation mirrors ok-ww Cartethyia.click_liberation():
// standard liberation cast with finishLiberationCast.
func cartethyiaClickLiberation(c combatActor) bool {
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

// cartethyiaClickResonance mirrors ok-ww Cartethyia.click_resonance():
// casts resonance while available for the given timeout.
func cartethyiaClickResonance(c combatActor, timeout time.Duration) bool {
	start := time.Now()
	clicked := false
	for c.resonanceAvailable() && time.Since(start) < timeout {
		if c.currentResonance() > 0 && c.forceSkill() {
			clicked = true
			c.state.cartethyiaResAt = time.Now()
			c.state.cartethyiaResFreeze = screenAnalyzer.FreezeDuration
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

// cartethyiaTryMidAirAttack mirrors ok-ww Cartethyia.try_mid_air_attack():
// jump+attack loop for up to 2s while mid-air-attack icon visible, or 0.8s fallback.
func cartethyiaTryMidAirAttack(c combatActor) {
	if !(c.state.cartethyiaTryMidAirOnce || c.cartethyiaMidAirAttackAvailable() || screenAnalyzer.Liberation || c.currentLiberation() > 0.05) {
		return
	}
	if c.cartethyiaMidAirAttackAvailable() {
		start := time.Now()
		for time.Since(start) < 2*time.Second {
			c.jump()
			c.sleep(100 * time.Millisecond)
			c.attack()
			c.sleep(100 * time.Millisecond)
			if !c.cartethyiaMidAirAttackAvailable() {
				c.sleep(400 * time.Millisecond)
				break
			}
		}
	} else if c.state.cartethyiaTryMidAirOnce {
		start := time.Now()
		for time.Since(start) < 800*time.Millisecond {
			c.jump()
			c.sleep(100 * time.Millisecond)
			c.attack()
			c.sleep(100 * time.Millisecond)
		}
	}
	c.state.cartethyiaTryMidAirOnce = false
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

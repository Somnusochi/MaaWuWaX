package combat

import "time"

// performCantarella mirrors ok-ww Cantarella.do_perform():
//
//	intro(1.2s attack, roccia/sanhua outro flag) → liberation →
//	mouse_forte_full or !forte_full: resonance → heavy_click_forte / echo / normal attack →
//	forte drain loop (held heavy, resonance, retry cycle) → echo → switch
func performCantarella(c combatActor) {
	performUnderOutro := false
	if c.recentlyIntroSwitchedIn(1700 * time.Millisecond) {
		c.attackFor(1200 * time.Millisecond)
		if c.switchedFromName("roccia", 1700*time.Millisecond) ||
			c.switchedFromName("sanhua", 1700*time.Millisecond) ||
			c.switchedFromName("sanhua2", 1700*time.Millisecond) {
			performUnderOutro = true
		}
	}
	cantarellaClickLiberation(c)
	if c.mouseForteFull() || !c.forteFull() {
		cantarellaSkill(c)
		if performUnderOutro && c.flying() {
			c.waitDown(1200 * time.Millisecond)
		}
		if !c.flying() && c.mouseForteFull() {
			if cantarellaHeavyClickForte(c) {
				c.state.lastHeavy = time.Now()
				c.state.cantarellaHeavyFreeze = screenAnalyzer.FreezeDuration
			}
		} else if c.echoWait(1 * time.Second) {
			c.requestSwitch()
			return
		} else {
			c.attackFor(100 * time.Millisecond)
			c.requestSwitch()
			return
		}
	}
	if !c.state.lastHeavy.IsZero() && c.freezeElapsed(c.state.lastHeavy, c.state.cantarellaHeavyFreeze) < 8*time.Second && !c.mouseForteFull() {
		if cantarellaDrainForte(c, performUnderOutro) && !performUnderOutro {
			c.requestSwitch()
			return
		}
	}
	if !c.echoWait(1 * time.Second) {
		c.attackFor(100 * time.Millisecond)
		c.requestSwitch()
		return
	}
	c.requestSwitch()
}

// cantarellaClickLiberation mirrors ok-ww Cantarella.click_liberation():
// casts liberation with up to 800ms wait, then finishLiberationCast.
func cantarellaClickLiberation(c combatActor) bool {
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

// cantarellaSkill mirrors ok-ww Cantarella.click_resonance():
// casts resonance while cantarella-specific resonance_available is true (up to 10s).
func cantarellaSkill(c combatActor) bool {
	if !cantarellaResonanceAvailable(c) {
		return false
	}
	start := time.Now()
	clicked := false
	for c.resonanceChainAvailable() && time.Since(start) < 10*time.Second {
		if c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

// cantarellaResonanceAvailable mirrors ok-ww Cantarella.resonance_available():
// Python: `not mouse_forte_full and is_forte_full -> not has_cd('resonance') else super`.
// BaseChar.super().resonance_available() is available('resonance', check_color=False),
// so both paths are CD-driven; the local debounce only prevents same-frame resends.
func cantarellaResonanceAvailable(c combatActor) bool {
	if !c.resonanceNoCD() {
		return false
	}
	return !cantarellaResonanceOnCooldown(c)
}

func cantarellaResonanceOnCooldown(c combatActor) bool {
	return c.freezeElapsed(c.state.lastResonance, c.state.lastResonanceFreeze) < 2*time.Second
}

// cantarellaHeavyClickForte mirrors ok-ww Cantarella.heavy_click_forte():
// holds heavy until mouse_forte_full becomes false.
func cantarellaHeavyClickForte(c combatActor) bool {
	if !c.mouseForteFull() {
		return false
	}
	c.holdHeavyUntil(2*time.Second, 100*time.Millisecond, func() bool {
		return !c.mouseForteFull()
	})
	return !c.mouseForteFull()
}

// cantarellaDrainForte mirrors ok-ww Cantarella's forte drain loop:
// held heavy + resonance retry cycle within 8s after last heavy.
func cantarellaDrainForte(c combatActor, performUnderOutro bool) bool {
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostTouchDown(0, 640, 360, 1).Wait()
	defer ctrl.PostTouchUp(0).Wait()

	forteDeadline := time.Now()
	retryThreshold := -100 * time.Millisecond

	for c.freezeElapsed(c.state.lastHeavy, c.state.cantarellaHeavyFreeze) < 8*time.Second && !c.mouseForteFull() {
		now := time.Now()
		if cantarellaResonanceAvailable(c) && cantarellaSkill(c) && !performUnderOutro {
			ctrl.PostTouchUp(0).Wait()
			return true
		}
		if !performUnderOutro && c.needFastPerform() && c.performElapsed() > 1100*time.Millisecond {
			break
		}
		if now.Sub(forteDeadline) > retryThreshold {
			ctrl.PostTouchUp(0).Wait()
			c.sleep(200 * time.Millisecond)
			ctrl.PostTouchDown(0, 640, 360, 1).Wait()
			retryThreshold += 1 * time.Second
		}
		if c.forteFull() {
			forteDeadline = now
		} else if now.Sub(forteDeadline) > 500*time.Millisecond {
			break
		}
		c.sleep(90 * time.Millisecond)
	}
	return false
}

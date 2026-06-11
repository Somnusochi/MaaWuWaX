package combat

import "time"

// performGalbrena mirrors ok-ww Galbrena.do_perform():
//
//	intro: mouse_down(1s)→need_fast→switch / +0.44s→mouse_up→right_click(0.6s) /
//	!intro+flying: wait_down → echo → forte_full+!fast: resonance→liberation→attack(1s) →
//	has_long_action+!fast: liberation→loop attack + shorekeeper_dodge → switch /
//	attack(1s)→resonance→switch
func performGalbrena(c combatActor) {
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		ctrl := c.ctx.GetTasker().GetController()
		ctrl.PostTouchDown(0, 640, 360, 1).Wait()
		c.sleep(1 * time.Second)
		if c.needFastPerform() {
			ctrl.PostTouchUp(0).Wait()
			c.requestSwitch()
			return
		}
		c.sleep(440 * time.Millisecond)
		ctrl.PostTouchUp(0).Wait()
		c.rightClickFor(600 * time.Millisecond)
	} else if c.flying() {
		c.waitDown(1200 * time.Millisecond)
	}

	c.echoImmediate()
	if c.forteFull() && !c.needFastPerform() {
		defaultClickResonance(c)
		if galbrenaClickLiberation(c) {
			c.attackFor(1 * time.Second)
		}
	}

	if c.hasLongAction() && !c.needFastPerform() {
		galbrenaClickLiberation(c)
		start := time.Now()
		for c.hasLongAction() && time.Since(start) < 10*time.Second {
			if c.flying() {
				shorekeeperAutoDodge(c, func() bool { return c.flying() })
			}
			c.attack()
			c.sleep(100 * time.Millisecond)
		}
		c.requestSwitch()
		return
	}
	c.attackFor(1 * time.Second)
	defaultClickResonance(c)
	c.requestSwitch()
}

// galbrenaClickLiberation mirrors ok-ww Galbrena.click_liberation():
// standard liberation cast with finishLiberationCast.
func galbrenaClickLiberation(c combatActor) bool {
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

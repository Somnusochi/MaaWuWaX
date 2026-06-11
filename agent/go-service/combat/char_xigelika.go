package combat

import (
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
)

// performXigelika mirrors ok-ww Xigelika.do_perform():
//
//	intro(0.77s attack) / wait_down → perform_everything loop:
//	  echo → flying→shorekeeper_dodge → handle_heavy → !intro→liberation(reset timeout) → resonance → attack → switch
func performXigelika(c combatActor) {
	if c.recentlyIntroSwitchedIn(1600 * time.Millisecond) {
		c.attackFor(770 * time.Millisecond)
	} else {
		c.waitDown(1200 * time.Millisecond)
	}
	start := time.Now()
	startFreeze := screenAnalyzer.FreezeDuration
	timeout := 500 * time.Millisecond
	if c.recentlyIntroSwitchedIn(2*time.Second) && c.switchedFromRole(roleSub, 2*time.Second) {
		timeout = 15 * time.Second
	}
	for c.freezeElapsed(start, startFreeze) < timeout {
		c.echo()
		if c.flying() {
			shorekeeperAutoDodge(c, func() bool { return c.flying() })
		}
		if xigelikaHandleHeavy(c) {
			c.requestSwitch()
			return
		}
		if !c.recentlyIntroSwitchedIn(1600*time.Millisecond) && xigelikaClickLiberation(c) {
			c.fBreak()
			timeout = 15 * time.Second
			start = time.Now()
			startFreeze = screenAnalyzer.FreezeDuration
			continue
		}
		if !xigelikaClickResonance(c) {
			c.attack()
		}
		c.sleep(50 * time.Millisecond)
	}
	c.requestSwitch()
}

func xigelikaClickLiberation(c combatActor) bool {
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

// xigelikaHandleHeavy mirrors ok-ww Xigelika.handle_heavy():
// loop while forte_full(3s): heavy_wait_highlight_down(mouse or hold_skill).
func xigelikaHandleHeavy(c combatActor) bool {
	handled := false
	start := time.Now()
	startFreeze := screenAnalyzer.FreezeDuration
	for c.forteFull() && c.freezeElapsed(start, startFreeze) < 3*time.Second {
		handled = true
		xigelikaHeavyWaitHighlightDown(c)
	}
	return handled
}

func xigelikaClickResonance(c combatActor) bool {
	start := time.Now()
	clicked := false
	for c.currentResonance() > 0.05 && time.Since(start) < 2*time.Second {
		if c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

// xigelikaHeavyWaitHighlightDown mirrors ok-ww Xigelika.heavy_wait_highlight_down():
// has_long_action→hold_heavy / else→hold_skill until forte_full drops.
func xigelikaHeavyWaitHighlightDown(c combatActor) bool {
	ctrl := c.ctx.GetTasker().GetController()
	if c.hasLongAction() {
		ctrl.PostTouchDown(0, 640, 360, 1).Wait()
		defer ctrl.PostTouchUp(0).Wait()
		deadline := time.Now().Add(1200 * time.Millisecond)
		for c.hasLongAction() && time.Now().Before(deadline) {
			c.sleep(50 * time.Millisecond)
		}
		c.sleep(10 * time.Millisecond)
		return true
	}
	if c.currentResonance() <= 0.05 {
		c.attack()
		c.sleep(50 * time.Millisecond)
		return false
	}
	ctrl.PostKeyDown(keycode.MustCode("E")).Wait()
	deadline := time.Now().Add(1200 * time.Millisecond)
	for c.forteFull() && time.Now().Before(deadline) {
		c.sleep(50 * time.Millisecond)
	}
	ctrl.PostKeyUp(keycode.MustCode("E")).Wait()
	c.sleep(10 * time.Millisecond)
	return true
}

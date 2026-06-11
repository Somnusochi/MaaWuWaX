package combat

import (
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
)

// performHiyuki mirrors ok-ww Hiyuki.do_perform():
//
//	intro(1s attack) → has_long_action2+lib_permission→hold_liberation →
//	has_long_action→perform_standard / has_long_action2→perform_lib → switch
func performHiyuki(c combatActor) {
	if c.state.lastPerform.IsZero() {
		c.state.libPermission = true
	}
	if c.recentlyIntroSwitchedIn(1500 * time.Millisecond) {
		c.attackFor(1 * time.Second)
	}
	if c.hasLongAction2() && c.state.libPermission && hiyukiLiberationAvailable(c) {
		hiyukiHoldLiberation(c)
	}
	if c.hasLongAction() && hiyukiLiberationNotGray(c) {
		performHiyukiStandard(c)
	}
	if c.hasLongAction2() {
		performHiyukiLiberationWindow(c)
	}
	c.requestSwitch()
}

func hiyukiLiberationAvailable(c combatActor) bool {
	return c.liberationAvailable()
}

func hiyukiLiberationNotGray(c combatActor) bool {
	return c.liberationNoCD()
}

func hiyukiTimeout(c combatActor) time.Duration {
	if c.recentlyIntroSwitchedIn(1500*time.Millisecond) && c.switchedFromName("linnai", 1500*time.Millisecond) {
		return 18 * time.Second
	}
	return 6 * time.Second
}

// performHiyukiStandard mirrors ok-ww Hiyuki.perform_standard():
// loop while has_long_action: echo→liberation / mouse_forte_full→right_click→heavy→wait_liberation→liberation / resonance+attack.
func performHiyukiStandard(c combatActor) {
	timeout := hiyukiTimeout(c)

	for c.hasLongAction() && c.performElapsed() < timeout {
		c.echoImmediate()
		if hiyukiLiberationAvailable(c) {
			if hiyukiClickLiberation(c) {
				c.state.libPermission = false
				return
			}
		}
		if c.mouseForteFull() {
			c.rightClick()
			c.sleep(50 * time.Millisecond)
			c.holdHeavyUntil(1200*time.Millisecond, 50*time.Millisecond, func() bool {
				return !c.mouseForteFull()
			})
			c.sleep(50 * time.Millisecond)
			c.state.lastHeavy = time.Now()
			waitStart := time.Now()
			for time.Since(waitStart) < 6*time.Second {
				if hiyukiLiberationAvailable(c) {
					break
				}
				c.attack()
				c.sleep(100 * time.Millisecond)
			}
			if hiyukiLiberationAvailable(c) {
				if hiyukiClickLiberation(c) {
					c.state.libPermission = false
					return
				}
			}
		}
		hiyukiClickResonance(c)
		c.attackFor(100 * time.Millisecond)
		c.sleep(50 * time.Millisecond)
	}
}

// performHiyukiLiberationWindow mirrors ok-ww Hiyuki.perform_lib():
// loop while has_long_action2: f_break→echo→lib_permission→hold_liberation /
//
//	resonance(exit if >2s) / lib_heavy→heavy_click→wait_liberation→hold /
//	left_prompt→click_loop / right_prompt→right_click / normal_attack.
func performHiyukiLiberationWindow(c combatActor) {
	start := time.Now()
	startFreeze := screenAnalyzer.FreezeDuration
	timeout := hiyukiTimeout(c)
	var lastRightClick time.Time

	for c.hasLongAction2() && c.performElapsed() < timeout {
		c.fBreak()
		c.echoImmediate()
		if c.state.libPermission && hiyukiLiberationAvailable(c) {
			hiyukiHoldLiberation(c)
			return
		}
		if hiyukiClickResonance(c) {
			if timeout == 6*time.Second && c.freezeElapsed(start, startFreeze) > 2*time.Second {
				return
			}
			c.attackFor(300 * time.Millisecond)
		} else if c.hiyukiLibHeavyReady() {
			c.holdHeavyUntil(700*time.Millisecond, 50*time.Millisecond, func() bool {
				return !c.hiyukiLibHeavyReady()
			})
			c.sleep(50 * time.Millisecond)
			c.state.lastHeavy = time.Now()
			c.state.libPermission = true
			waitDeadline := time.Now().Add(500 * time.Millisecond)
			for time.Now().Before(waitDeadline) && !hiyukiLiberationAvailable(c) {
				c.attack()
				c.sleep(100 * time.Millisecond)
			}
			if hiyukiLiberationAvailable(c) {
				hiyukiHoldLiberation(c)
			}
			return
		} else if c.hiyukiLeftPrompt() {
			leftStart := time.Now()
			for c.hiyukiLeftPrompt() && time.Since(leftStart) < 3*time.Second {
				c.attack()
				c.sleep(100 * time.Millisecond)
			}
			if timeout == 6*time.Second && c.freezeElapsed(start, startFreeze) > 2*time.Second {
				return
			}
			c.sleep(100 * time.Millisecond)
		} else if c.hiyukiRightPrompt() {
			if lastRightClick.IsZero() || time.Since(lastRightClick) >= 1*time.Second {
				c.rightClick()
				lastRightClick = time.Now()
			}
			c.sleep(100 * time.Millisecond)
		} else {
			c.attack()
			c.sleep(100 * time.Millisecond)
		}
		c.sleep(50 * time.Millisecond)
	}
}

// hiyukiClickResonance mirrors ok-ww Hiyuki.click_resonance(send_click=False, time_out=0):
// casts resonance while available for up to 15s.
func hiyukiClickResonance(c combatActor) bool {
	start := time.Now()
	clicked := false
	for c.resonanceAvailable() && time.Since(start) < 15*time.Second {
		if c.currentResonance() > 0 && c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

// hiyukiClickLiberation mirrors ok-ww Hiyuki.click_liberation():
// casts liberation with animation wait; returns false if still available after cast.
func hiyukiClickLiberation(c combatActor) bool {
	if !c.param.UseLiberation {
		return false
	}
	start := time.Now()
	clicked := false
	for time.Since(start) < 800*time.Millisecond && hiyukiLiberationAvailable(c) {
		c.forceLiberation()
		clicked = true
		c.sleep(100 * time.Millisecond)
	}
	if !clicked {
		retryDeadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(retryDeadline) && hiyukiLiberationNotGray(c) {
			if c.forceLiberation() {
				clicked = true
			}
			c.sleep(50 * time.Millisecond)
		}
		if !clicked {
			return false
		}
	}
	leaveDeadline := time.Now().Add(400 * time.Millisecond)
	for time.Now().Before(leaveDeadline) && c.isCurrentChar() {
		c.sleep(50 * time.Millisecond)
	}
	if c.isCurrentChar() || hiyukiLiberationAvailable(c) {
		return false
	}
	freezeStart := time.Now()
	backDeadline := time.Now().Add(7 * time.Second)
	for time.Now().Before(backDeadline) && !c.isCurrentChar() {
		c.sleep(50 * time.Millisecond)
	}
	if !c.isCurrentChar() {
		return false
	}
	c.addFreezeDuration(time.Since(freezeStart))
	c.state.lastLiberation = time.Now()
	return true
}

// hiyukiHoldLiberation mirrors ok-ww Hiyuki.hold_liberation():
// holds liberation key while in_team and liberation_available (up to 8s), waits for return.
func hiyukiHoldLiberation(c combatActor) bool {
	if !c.param.UseLiberation {
		return false
	}
	start := time.Now()
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostKeyDown(keycode.MustCode("R")).Wait()
	defer ctrl.PostKeyUp(keycode.MustCode("R")).Wait()
	clicked := false
	for time.Since(start) < 8*time.Second && c.isCurrentChar() && hiyukiLiberationAvailable(c) {
		clicked = true
		c.sleep(50 * time.Millisecond)
	}
	if !clicked {
		return false
	}
	leaveDeadline := time.Now().Add(400 * time.Millisecond)
	for time.Now().Before(leaveDeadline) && c.isCurrentChar() {
		c.sleep(50 * time.Millisecond)
	}
	if c.isCurrentChar() || hiyukiLiberationAvailable(c) {
		return false
	}
	backDeadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(backDeadline) && !c.isCurrentChar() {
		c.sleep(50 * time.Millisecond)
	}
	if !c.isCurrentChar() {
		return false
	}
	c.addFreezeDuration(time.Since(start))
	return true
}

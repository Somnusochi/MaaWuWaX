package combat

import (
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
)

// performAugusta mirrors ok-ww Augusta.do_perform():
//
//	intro(1.13s attack) → majesty/prowess dual-state loop within timeOut+3s window:
//	  check_majesty → perform_majesty(echo→switch) /
//	  flying → shorekeeper_auto_dodge /
//	  prowess → perform_prowess(heavy) /
//	  resonance → prowess or majesty chain /
//	  liberation → switch if timeOut < 14
//	fallthrough: echo → switch
func performAugusta(c combatActor) {
	timeOut := 3 * time.Second
	if c.recentlyIntroSwitchedIn(1700 * time.Millisecond) {
		c.attackFor(1130 * time.Millisecond)
		if c.switchedFromName("iuno", 1700*time.Millisecond) {
			timeOut = 14 * time.Second
		}
	}
	if c.flying() {
		c.waitDown(2500 * time.Millisecond)
	}
	start := time.Now()
	loopWindow := timeOut + 3*time.Second
	for time.Since(start) < loopWindow {
		if c.augustaMajestyReady() {
			if augustaPerformMajesty(c) {
				c.echo()
				c.requestSwitch()
				return
			}
		}
		if c.flying() {
			shorekeeperAutoDodge(c, func() bool { return c.flying() })
		}
		if c.augustaProwessReady() && augustaPerformProwess(c) {
			if time.Since(start) > timeOut {
				c.requestSwitch()
				return
			}
		}
		if augustaResonanceAvailable(c) {
			clicked, duration := augustaClickResonance(c)
			if clicked {
				if duration < 1400*time.Millisecond {
					if c.flying() {
						continue
					}
					prowessDeadline := time.Now().Add(1 * time.Second)
					for time.Now().Before(prowessDeadline) {
						if c.augustaProwessReady() && augustaPerformProwess(c) {
							if time.Since(start) > timeOut && !c.flying() {
								c.requestSwitch()
								return
							}
							break
						}
						c.sleep(100 * time.Millisecond)
					}
				} else if c.augustaMajestyReady() {
					c.waitDown(1200 * time.Millisecond)
					if augustaPerformMajesty(c) {
						c.echo()
					}
					c.requestSwitch()
					return
				}
			}
		}
		if c.augustaLibReady() && augustaPerformLiberation(c) {
			if timeOut < 14*time.Second {
				c.requestSwitch()
				return
			}
		}
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	c.echo()
	c.requestSwitch()
}

// augustaResonanceAvailable mirrors ok-ww Augusta.resonance_available():
// not has_cd('resonance'), with local debounce to stop repeated sends between
// analyzer frames and while higher-priority Augusta follow-up prompts are visible.
func augustaResonanceAvailable(c combatActor) bool {
	if !c.resonanceNoCD() {
		return false
	}
	if c.freezeElapsed(c.state.lastResonance, c.state.lastResonanceFreeze) < 2*time.Second {
		return false
	}
	return !c.augustaProwessReady() && !c.augustaMajestyReady()
}

// augustaClickResonance mirrors ok-ww Augusta.click_resonance():
// casts resonance and returns (clicked, duration).
func augustaClickResonance(c combatActor) (bool, time.Duration) {
	start := time.Now()
	clicked := false
	for augustaResonanceAvailable(c) && time.Since(start) < 15*time.Second {
		if c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked, time.Since(start)
}

// augustaPerformProwess mirrors ok-ww Augusta.perform_prowess():
// heavy_click_forte → normal attack 0.3s.
func augustaPerformProwess(c combatActor) bool {
	if !c.augustaProwessReady() {
		return false
	}
	c.holdHeavyUntil(1200*time.Millisecond, 100*time.Millisecond, func() bool {
		return !c.augustaProwessReady()
	})
	c.attackFor(300 * time.Millisecond)
	return true
}

// augustaPerformMajesty mirrors ok-ww Augusta.perform_majesty():
// holds liberation key to enter majesty animation, waits for return, records freeze.
func augustaPerformMajesty(c combatActor) bool {
	if !c.augustaMajestyReady() {
		return false
	}
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostKeyDown(keycode.MustCode("R")).Wait()
	start := time.Now()
	leaveDeadline := start.Add(600 * time.Millisecond)
	if c.flying() {
		leaveDeadline = start.Add(200 * time.Millisecond)
	}
	for time.Now().Before(leaveDeadline) && c.isCurrentChar() {
		c.sleep(50 * time.Millisecond)
	}
	ctrl.PostKeyUp(keycode.MustCode("R")).Wait()
	if c.isCurrentChar() {
		return false
	}
	freezeStart := time.Now()
	backDeadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(backDeadline) && !c.isCurrentChar() {
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	if !c.isCurrentChar() {
		return false
	}
	c.addFreezeDuration(time.Since(freezeStart))
	c.state.lastLiberation = time.Now()
	return true
}

// augustaPerformLiberation mirrors ok-ww Augusta.click_liberation():
// casts liberation while augustaLibReady for up to 2s.
func augustaPerformLiberation(c combatActor) bool {
	start := time.Now()
	clicked := false
	for c.augustaLibReady() && time.Since(start) < 2*time.Second {
		c.forceLiberation()
		clicked = true
		c.sleep(100 * time.Millisecond)
	}
	if clicked && !c.augustaLibReady() {
		c.state.lastLiberation = time.Now()
		return true
	}
	return false
}

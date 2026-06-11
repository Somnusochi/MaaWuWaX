package combat

import (
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/mouse"
)

// performJiyan mirrors ok-ww Jiyan.do_perform():
//
//	intro(2s attack) → liberation: resonance+middle_click+normal_attack 12s loop → switch /
//	!liberation: heavy+resonance/echo cycle till forte_full or con_full → resonance→echo→switch
func performJiyan(c combatActor) {
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		c.attackFor(2 * time.Second)
	}
	if jiyanClickLiberation(c) {
		deadline := time.Now().Add(12 * time.Second)
		for time.Now().Before(deadline) {
			if jiyanClickResonance(c, 500*time.Millisecond) {
				mouse.MiddleClick(c.ctx.GetTasker().GetController())
			}
			c.attack()
			c.sleep(80 * time.Millisecond)
		}
		c.requestSwitch()
		return
	}
	i := 0
	start := time.Now()
	for !c.forteFull() && screenAnalyzer.ConcertoPct < 1.0 && time.Since(start) < 10*time.Second {
		if i%4 == 0 {
			c.heavy(600 * time.Millisecond)
			if c.currentResonance() > 0.05 || c.currentEcho() > 0.05 {
				mouse.MiddleClick(c.ctx.GetTasker().GetController())
				break
			}
			i = 0
		}
		c.attack()
		c.sleep(90 * time.Millisecond)
		i++
	}
	if !c.forteFull() && jiyanClickResonance(c, 800*time.Millisecond) {
		c.sleep(1 * time.Second)
	}
	if c.echoWait(1 * time.Second) {
		c.requestSwitch()
		return
	}
	c.requestSwitch()
}

// jiyanClickLiberation mirrors ok-ww Jiyan.click_liberation():
// standard liberation cast with finishLiberationCast.
func jiyanClickLiberation(c combatActor) bool {
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

// jiyanClickResonance mirrors ok-ww Jiyan.click_resonance():
// casts resonance while available for the given timeout.
func jiyanClickResonance(c combatActor, timeout time.Duration) bool {
	if c.currentResonance() <= 0.05 {
		return false
	}
	start := time.Now()
	clicked := false
	for c.currentResonance() > 0.05 && time.Since(start) < timeout {
		if c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

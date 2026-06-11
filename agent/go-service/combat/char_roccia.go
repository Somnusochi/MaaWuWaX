package combat

import (
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
)

func performRoccia(c combatActor) {
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		c.heavy(1600 * time.Millisecond)
		c.sleep(100 * time.Millisecond)
		rocciaPlunge(c)
		if c.currentLiberation() <= 0.05 && c.currentResonance() <= 0.05 {
			// KNOWN_DIFF: Python uses CD-based availability check; Go uses UI energy check (semantically equivalent for skill-available guard)
			rocciaRequestSwitch(c)
			return
		}
	}
	liberated := rocciaClickLiberation(c)
	if rocciaClickResonance(c) || !liberated {
		rocciaPlunge(c)
		rocciaRequestSwitch(c)
		return
	}
	c.echoWait(1 * time.Second)
	rocciaRequestSwitch(c)
}

// rocciaPlunge mirrors ok-ww Roccia.plunge():
// holds W + click loop for up to 6s while mouseForteFull,
// retries liberation+resonance after 2s if both come off cooldown.
func rocciaPlunge(c combatActor) {
	if c.needFastPerform() {
		c.attack()
		for c.performElapsed() < 1100*time.Millisecond {
			c.attack()
			c.sleep(100 * time.Millisecond)
		}
		return
	}
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostKeyDown(keycode.MustCode("W")).Wait()
	defer ctrl.PostKeyUp(keycode.MustCode("W")).Wait()
	start := time.Now()
	for c.mouseForteFull() && time.Since(start) < 6*time.Second {
		if time.Since(start) > 2*time.Second &&
			c.currentResonance() > 0.05 &&
			c.currentLiberation() > 0.05 {
			// KNOWN_DIFF: Python uses CD-based availability check; Go uses UI energy check (semantically equivalent for skill-available guard)
			if rocciaClickLiberation(c) {
				rocciaClickResonance(c)
				start = time.Now() // reset timer on successful lib+res
				continue
			}
		}
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
}

func rocciaClickLiberation(c combatActor) bool {
	if !c.param.UseLiberation || (!screenAnalyzer.Liberation && c.currentLiberation() <= 0.05) {
		return false
	}
	start := time.Now()
	clicked := false
	for time.Since(start) < 15*time.Second && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) {
		c.forceLiberation()
		clicked = true
		c.sleep(100 * time.Millisecond)
	}
	return finishLiberationCast(c, clicked, 3*time.Second)
}

func rocciaClickResonance(c combatActor) bool {
	if c.currentResonance() <= 0.05 {
		return false
	}
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

func rocciaRequestSwitch(c combatActor) {
	// ok-ww can hand a tool_box payload to the next character on switch. The
	// current Go combat framework only exposes base switch metadata handled by
	// requestSwitch() (source slot, intro timing, buff refresh) and has no
	// per-switch payload channel we can safely set from this file alone.
	c.requestSwitch()
}

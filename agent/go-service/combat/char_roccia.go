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
		if !rocciaLiberationAvailable(c) && !rocciaResonanceAvailable(c) {
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
			c.resonanceNoCD() &&
			c.liberationNoCD() {
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
	if !rocciaLiberationAvailable(c) {
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
	if !rocciaResonanceAvailable(c) {
		return false
	}
	start := time.Now()
	clicked := false
	for c.resonanceChainAvailable() && time.Since(start) < 15*time.Second {
		if c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

func rocciaLiberationAvailable(c combatActor) bool {
	return c.param.UseLiberation && c.liberationNoCD() && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05)
}

func rocciaResonanceAvailable(c combatActor) bool {
	if !c.resonanceNoCD() {
		return false
	}
	return c.freezeElapsed(c.state.lastResonance, c.state.lastResonanceFreeze) >= 2*time.Second
}

func rocciaRequestSwitch(c combatActor) {
	c.state.grantToolBoxOnIntro = true
	c.requestSwitch()
}

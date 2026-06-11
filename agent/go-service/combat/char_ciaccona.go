package combat

import (
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
)

// performCiaccona mirrors ok-ww Ciaccona.do_perform():
//
//	intro(0.8s attack + 0.7s if !fast_perform) → echo(if current_echo<0.22) →
//	!intro: jump_with_click(0.4s)+wait_down+normal attack → resonance (jump=false, wait=true) →
//	forte>=3: jump→heavy_click_forte → liberation (wait+attr2 continues_click_a) → echo → switch
func performCiaccona(c combatActor) {
	c.state.ciacconaInLiberation = false
	waitAfterAction := false
	jumpNeeded := true
	attr := c.ciacconaAttribute()
	hasIntro := c.recentlyIntroSwitchedIn(1700 * time.Millisecond)
	fastPerform := ciacconaNeedFastPerform(c)

	if hasIntro {
		c.attackFor(800 * time.Millisecond)
		if !fastPerform {
			c.attackFor(700 * time.Millisecond)
		}
	}
	if c.currentEcho() < 0.22 {
		c.echoImmediate()
	}
	if !hasIntro {
		if !fastPerform && !c.mouseForteFull() {
			ciacconaJumpWithClick(c, 400*time.Millisecond)
			c.waitDown(1200 * time.Millisecond)
			c.attackFor(200 * time.Millisecond)
		}
	}

	if ciacconaClickResonance(c) {
		waitAfterAction = true
		jumpNeeded = false
	}
	if c.mouseForteFull() || screenAnalyzer.CiacconaForte >= 3 {
		if jumpNeeded && !c.flying() {
			jumpDeadline := time.Now().Add(300 * time.Millisecond)
			for !c.flying() && time.Now().Before(jumpDeadline) {
				c.jump()
				c.sleep(10 * time.Millisecond)
			}
		}
		c.holdHeavyUntil(2*time.Second, 50*time.Millisecond, func() bool {
			return !c.mouseForteFull()
		})
		waitAfterAction = true
	}
	// Mirror ok-ww: self.liberation_available() guard wraps the 0.4s wait.
	// Only wait when liberation is actually available; otherwise skip the sleep.
	if waitAfterAction && c.liberationAvailable() {
		c.sleep(400 * time.Millisecond)
	}
	if ciacconaClickLiberation(c) {
		c.state.ciacconaInLiberation = true
		if attr == 2 {
			ciacconaContinuesA(c, 600*time.Millisecond)
		}
	}
	if !c.state.ciacconaInLiberation && c.currentEcho() > 0.25 {
		c.echoWait(1 * time.Second)
	}
	c.requestSwitch()
}

// ciacconaClickLiberation mirrors ok-ww Ciaccona.click_liberation():
// standard liberation cast with finishLiberationCast.
func ciacconaClickLiberation(c combatActor) bool {
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

// ciacconaClickResonance mirrors ok-ww Ciaccona.click_resonance():
// casts resonance while available for up to 10s.
func ciacconaClickResonance(c combatActor) bool {
	start := time.Now()
	clicked := false
	for c.resonanceAvailable() && time.Since(start) < 10*time.Second {
		if c.currentResonance() > 0 && c.forceSkill() {
			clicked = true
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

// ciacconaJumpWithClick mirrors ok-ww Ciaccona.click_jump_with_click():
// alternates attack/jump for duration, starting with attack.
func ciacconaJumpWithClick(c combatActor, duration time.Duration) {
	start := time.Now()
	click := true
	for time.Since(start) < duration {
		if click {
			c.attack()
		} else {
			c.jump()
		}
		click = !click
		c.sleep(100 * time.Millisecond)
	}
}

// ciacconaContinuesA mirrors ok-ww Ciaccona.continues_click_a():
// sends A key for the given duration.
func ciacconaContinuesA(c combatActor, duration time.Duration) {
	deadline := time.Now().Add(duration)
	ctrl := c.ctx.GetTasker().GetController()
	for time.Now().Before(deadline) {
		ctrl.PostClickKey(keycode.MustCode("A")).Wait()
		c.sleep(50 * time.Millisecond)
	}
}

func ciacconaNeedFastPerform(c combatActor) bool {
	return c.teamHas("cartethyia") && c.cartethyiaIsSmall()
}

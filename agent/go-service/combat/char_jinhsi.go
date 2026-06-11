package combat

import "time"

// performJinhsi mirrors ok-ww Jinhsi.do_perform():
//
//	incarnation→handle_incarnation(resonance+attack+echo)→switch /
//	intro/incarnation_cd→handle_intro(resonance_spam→cd_check→liberation→attack)→switch /
//	default→echo→switch
func performJinhsi(c combatActor) {
	if c.state.incarnationActive {
		c.state.incarnationActive = false
		performJinhsiIncarnation(c)
		c.requestSwitch()
		return
	}

	if c.recentlyIntroSwitchedIn(1600*time.Millisecond) || c.state.incarnationCD {
		startedIncarnation := performJinhsiIntro(c)
		c.state.incarnationActive = startedIncarnation
		if startedIncarnation {
			c.state.incarnationCD = false
		}
		c.requestSwitch()
		return
	}

	// Outside the special window, ok-ww Jinhsi mostly just dumps echo and leaves.
	c.echoWait(1 * time.Second)
	c.requestSwitch()
}

// performJinhsiIntro mirrors ok-ww Jinhsi.handle_intro():
// resonance spam(1.5s) → cd check(0.3s-1.5s→incarnation_cd+echo) /
//
//	resonance_finished→liberation+attack(0.3s) or attack(1.4s)→returns incarnation=true.
func performJinhsiIntro(c combatActor) bool {
	start := time.Now()
	startFreeze := screenAnalyzer.FreezeDuration
	deadline := 1500 * time.Millisecond

	for c.freezeElapsed(start, startFreeze) < deadline {
		elapsed := c.freezeElapsed(start, startFreeze)
		if !jinhsiIntroResonanceAvailable(c) {
			if elapsed > 300*time.Millisecond && elapsed < 1500*time.Millisecond && jinhsiIntroResonanceOnCooldown(c) {
				c.state.incarnationCD = true
				if !c.echoWait(1 * time.Second) {
					c.attack()
					c.sleep(100 * time.Millisecond)
				}
				return false
			}
			if elapsed >= 1500*time.Millisecond {
				break
			}
		} else if c.forceSkill() {
			c.sleep(90 * time.Millisecond)
			continue
		}
		c.sleep(60 * time.Millisecond)
	}

	if jinhsiClickLiberation(c) {
		c.attackFor(300 * time.Millisecond)
	} else {
		c.attackFor(1400 * time.Millisecond)
	}

	c.state.incarnationCD = false
	c.state.jinhsiIncarnationUntil = time.Now().Add(6 * time.Second)
	return true
}

// jinhsiIntroResonanceAvailable mirrors the intro-only ok-ww availability check more
// closely than a raw gauge threshold by requiring both visible resonance resource and
// a cleared local resonance cooldown anchor.
func jinhsiIntroResonanceAvailable(c combatActor) bool {
	return c.currentResonance() > 0.05 && !jinhsiIntroResonanceOnCooldown(c)
}

func jinhsiIntroResonanceOnCooldown(c combatActor) bool {
	return c.freezeElapsed(c.state.lastResonance, c.state.lastResonanceFreeze) < 2*time.Second
}

// jinhsiClickLiberation mirrors ok-ww Jinhsi.click_liberation(send_click=True):
// standard liberation cast with finishLiberationCast.
func jinhsiClickLiberation(c combatActor) bool {
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

// performJinhsiIncarnation mirrors ok-ww Jinhsi.handle_incarnation():
// alternating resonance+attack(6s timeout) → echo/click → record animation freeze.
func performJinhsiIncarnation(c combatActor) {
	start := time.Now()
	var animationStarted time.Time
	useResonance := true
	for time.Since(start) < 6*time.Second {
		if c.isCurrentChar() {
			if useResonance {
				c.forceSkill()
			} else {
				c.attack()
			}
			useResonance = !useResonance
			if !animationStarted.IsZero() {
				break
			}
		} else {
			if animationStarted.IsZero() {
				animationStarted = time.Now()
			}
		}
		c.sleep(100 * time.Millisecond)
	}
	if !animationStarted.IsZero() {
		c.addFreezeDuration(time.Since(animationStarted))
	}

	if !c.echoWait(1 * time.Second) {
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
}

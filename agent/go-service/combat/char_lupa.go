package combat

import "time"

// performLupa mirrors ok-ww Lupa.do_perform():
//
//	intro(1s attack, changli outro flag) → echo → res_wolf(!outro→switch) →
//	forte=2: drain_forte(heavy/attack until not full)→wolf=true → outro→wait_wolf+res_wolf →
//	resonance(clear liberation timer)→liberation→wait_down / switch →
//	f_break→liberation→attack→still_in_liberation: jump_click(4s)+drain_forte+wolf+outro → switch
func performLupa(c combatActor) {
	inOutro := false
	if c.recentlyIntroSwitchedIn(1600 * time.Millisecond) {
		c.attackFor(1 * time.Second)
		inOutro = c.switchedFromName("changli", 2*time.Second) || c.switchedFromName("changli2", 2*time.Second) || c.switchedFromName("chang_changli", 2*time.Second)
	}
	c.echo()

	if lupaResWolf(c) && !inOutro {
		c.requestSwitch()
		return
	}

	if lupaJudgeForte(c) == 2 && !c.lupaWolfReady() {
		if c.flying() {
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) && c.forteFull() {
				c.attack()
				c.sleep(100 * time.Millisecond)
			}
		} else {
			c.heavy(600 * time.Millisecond)
			deadline := time.Now().Add(1400 * time.Millisecond)
			for time.Now().Before(deadline) && c.forteFull() {
				c.attack()
				c.sleep(100 * time.Millisecond)
			}
		}
		if !c.forteFull() || c.lupaWolfReady() {
			c.state.wolfReady = true
		}
		if !inOutro {
			c.requestSwitch()
			return
		}
		lupaWaitWolfReady(c, 500*time.Millisecond)
		lupaResWolf(c)
	}
	if c.currentResonance() > 0.05 && lupaClickResonance(c) {
		c.state.lastLiberation = time.Time{}
		c.state.lupaLiberationFreeze = 0
		if lupaLiberationAvailable(c) {
			c.waitDown(1200 * time.Millisecond)
		} else {
			c.requestSwitch()
			return
		}
	}
	c.fBreak()
	if (inOutro || !c.needFastPerform()) && lupaClickLiberation(c) {
		c.attackFor(300 * time.Millisecond)
		if inOutro {
			c.attackFor(1 * time.Second)
		} else {
			c.requestSwitch()
			return
		}
	}
	if lupaStillInLiberation(c) {
		lupaJumpWithClick(c, 4*time.Second)
		if c.flying() {
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) && c.forteFull() {
				c.attack()
				c.sleep(100 * time.Millisecond)
			}
		} else {
			c.heavy(600 * time.Millisecond)
			deadline := time.Now().Add(1400 * time.Millisecond)
			for time.Now().Before(deadline) && c.forteFull() {
				c.attack()
				c.sleep(100 * time.Millisecond)
			}
		}
		if !c.forteFull() {
			c.state.wolfReady = true
		}
		if inOutro {
			lupaWaitWolfReady(c, 500*time.Millisecond)
			lupaResWolf(c)
		}
		c.requestSwitch()
		return
	}
	c.attackFor(100 * time.Millisecond)
	c.requestSwitch()
}

func lupaStillInLiberation(c combatActor) bool {
	return !c.state.lastLiberation.IsZero() && c.freezeElapsed(c.state.lastLiberation, c.state.lupaLiberationFreeze) < 12*time.Second
}

func lupaJudgeForte(c combatActor) int {
	if !c.forteFull() {
		return 0
	}
	if screenAnalyzer.LupaForte > 0 {
		return screenAnalyzer.LupaForte
	}
	if c.mouseForteFull() {
		return 2
	}
	return 0
}

func lupaJumpWithClick(c combatActor, delay time.Duration) {
	start := time.Now()
	clickJump := false
	for time.Since(start) < delay {
		if clickJump {
			c.attack()
		} else {
			c.jump()
		}
		clickJump = !clickJump
		if lupaJudgeForte(c) == 2 {
			return
		}
		c.sleep(100 * time.Millisecond)
	}
}

func lupaClickResonance(c combatActor) bool {
	if !lupaResonanceAvailable(c) {
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

func lupaClickLiberation(c combatActor) bool {
	if !lupaLiberationAvailable(c) {
		return false
	}
	start := time.Now()
	clicked := false
	for time.Since(start) < 800*time.Millisecond && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) {
		c.forceLiberation()
		clicked = true
		c.sleep(100 * time.Millisecond)
	}
	if !clicked {
		retryDeadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(retryDeadline) && c.currentLiberation() > 0.001 {
			c.forceLiberation()
			clicked = true
			c.sleep(100 * time.Millisecond)
		}
	}
	if !finishLiberationCast(c, clicked, 7*time.Second) {
		return false
	}
	c.state.lupaLiberationFreeze = screenAnalyzer.FreezeDuration
	return true
}

func lupaResonanceAvailable(c combatActor) bool {
	if !c.resonanceNoCD() {
		return false
	}
	return c.freezeElapsed(c.state.lastResonance, c.state.lastResonanceFreeze) >= 2*time.Second
}

func lupaLiberationAvailable(c combatActor) bool {
	return c.param.UseLiberation && c.liberationNoCD() && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05)
}

func lupaWaitWolfReady(c combatActor, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c.lupaWolfReady() {
			return true
		}
		c.sleep(50 * time.Millisecond)
	}
	return c.lupaWolfReady()
}

// lupaResWolf mirrors ok-ww Lupa.res_wolf():
// casts resonance while wolf_icon2 visible, clears liberation timer, sleeps 1.2s.
func lupaResWolf(c combatActor) bool {
	if !c.state.wolfReady && !c.lupaWolfReady() {
		return false
	}
	start := time.Now()
	clicked := false
	for time.Since(start) < 1*time.Second {
		if c.state.wolfReady && time.Since(start) < 200*time.Millisecond {
			// brief grace window after draining forte before the wolf icon appears
		} else if !c.lupaWolfReady() {
			break
		}
		c.forceSkill()
		clicked = true
		c.sleep(100 * time.Millisecond)
		if c.currentResonance() <= 0.05 {
			break
		}
	}
	if clicked {
		c.state.lastLiberation = time.Time{}
		c.state.lupaLiberationFreeze = 0
		c.state.wolfReady = false
		c.sleep(1200 * time.Millisecond)
	}
	return clicked
}

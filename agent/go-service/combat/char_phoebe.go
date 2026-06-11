package combat

import "time"

type phoebeActionState int

const (
	phoebeActionUnavailable phoebeActionState = iota
	phoebeActionSuccess
	phoebeActionTimeout
)

// performPhoebe mirrors ok-ww Phoebe.do_perform():
//
//	dual-attribute system (light=1/blue=2). intro(1.5s attack) →
//	attribute_mismatch→cast_remaining_skills → zani_linkage →
//	absolution_or_confession(hold skill/heavy, starflash_combo) →
//	liberation(send_click) → starflash_combo → resonance(once vs full) → switch
func performPhoebe(c combatActor) {
	turnStart := time.Now()
	c.state.phoebeLastOutroAt = time.Time{}
	if !c.phoebeStarVisible() {
		c.state.phoebeStarLatched = false
	}

	preferredSupport := c.phoebePreferredSupport()
	if c.recentlyIntroSwitchedIn(1700 * time.Millisecond) {
		c.attackFor(1500 * time.Millisecond)
	} else {
		c.sleep(10 * time.Millisecond)
	}

	currentSupport := c.phoebeSupportMode()
	if !preferredSupport {
		c.echoImmediate()
	}
	if c.flying() {
		c.attackFor(100 * time.Millisecond)
		phoebePrepareSwitch(c, preferredSupport)
		c.requestSwitch()
		return
	}

	attributeMismatch := c.phoebeStarAvailable() && preferredSupport != currentSupport
	if attributeMismatch {
		phoebeCastRemainingSkills(c, currentSupport, false)
	}

	if preferredSupport && c.teamHasAny("zani", "zani2") {
		if !c.phoebeStarAvailable() {
			phoebeAbsolutionOrConfession(c, preferredSupport)
		}
		if phoebeZaniLinkage(c, preferredSupport) {
			phoebePrepareSwitch(c, preferredSupport)
			c.requestSwitch()
			return
		}
	}

	waitUI := 350*time.Millisecond - time.Since(turnStart)
	if waitUI > 0 && c.phoebeStarAvailable() && !phoebeHasForte(c) {
		c.attackFor(waitUI)
	}

	statusEntered := phoebeAbsolutionOrConfession(c, preferredSupport)
	if (!attributeMismatch || statusEntered == phoebeActionSuccess) && c.phoebeStarAvailable() && phoebeClickLiberation(c, true) {
		c.state.phoebeLiberationCount++
	}
	if statusEntered == phoebeActionSuccess || phoebeHasForte(c) {
		phoebeStarflashCombo(c, preferredSupport)
	}
	if phoebeResonanceAvailable(c) {
		if preferredSupport {
			phoebeClickResonanceOnce(c)
		} else {
			phoebeClickResonance(c, true)
		}
		phoebePrepareSwitch(c, preferredSupport)
		c.requestSwitch()
		return
	}
	phoebePrepareSwitch(c, preferredSupport)
	c.attackFor(100 * time.Millisecond)
	c.requestSwitch()
}

func phoebePrepareSwitch(c combatActor, support bool) {
	phoebeRecordPreSwitchState(c, support)
	if support && screenAnalyzer.ConcertoPct >= 1.0 {
		// Keep the pre-switch echo close to ok-ww, but leave outro count/timestamp
		// recording to requestSwitch() so a failed target pick does not leak state.
		c.echoWait(1 * time.Second)
	}
}

func phoebeRecordPreSwitchState(c combatActor, support bool) {
	if c.phoebeStarVisible() {
		c.state.phoebeStarLatched = true
	} else if phoebeForteEmpty(c) && !c.phoebeConfessionReady() {
		c.state.phoebeStarLatched = false
	}

	if !support {
		return
	}

	// Keep the support-chain counters in a minimally self-consistent shape before
	// leaving the field so the next selector tick reads the latest local progress.
	if c.state.phoebeStarflashCount > 0 && c.state.phoebeEnterStatusCount == 0 {
		c.state.phoebeEnterStatusCount = 1
	}
	if c.state.phoebeLiberationCount > 0 && c.state.phoebeEnterStatusCount == 0 {
		c.state.phoebeEnterStatusCount = 1
	}
}

func phoebeCastRemainingSkills(c combatActor, support bool, liber bool) time.Time {
	start := time.Time{}
	skillCount := 4
	if support {
		skillCount = 2
	}
	for range skillCount {
		if liber && c.state.phoebeLiberationCount < 1 && phoebeLiberationAvailable(c) {
			if phoebeClickLiberation(c, false) {
				c.state.phoebeLiberationCount++
			}
		}
		if phoebeHasForte(c) {
			phoebeStarflashCombo(c, support)
			start = time.Now()
		}
	}
	return start
}

func phoebeHasForte(c combatActor) bool {
	return phoebeForteTier(c) > 0 || c.forteFull()
}

func phoebeForteEmpty(c combatActor) bool {
	return phoebeForteTier(c) == 0 && c.currentForte() <= 0.01
}

func phoebeForteTier(c combatActor) int {
	if c.phoebePreferredSupport() {
		return screenAnalyzer.PhoebeBlueForte
	}
	return screenAnalyzer.PhoebeLightForte
}

func phoebeForteFull(c combatActor, support bool) bool {
	if !c.phoebeStarAvailable() {
		return c.forteFull()
	}
	if support {
		return screenAnalyzer.PhoebeFullBlue || c.forteFull()
	}
	return screenAnalyzer.PhoebeFullLight || c.forteFull()
}

func phoebeResetAction(c combatActor, support bool) {
	if !support {
		return
	}
	c.state.phoebeEnterStatusCount = 0
	c.state.phoebeLiberationCount = 0
	c.state.phoebeStarflashCount = 0
	c.state.phoebeOutroCount = 0
}

// phoebeAbsolutionOrConfession mirrors ok-ww Phoebe.absolution_or_confession():
// holds skill(support=true) or heavy(support=false) while condition met, right_click finish.
func phoebeAbsolutionOrConfession(c combatActor, support bool) phoebeActionState {
	condition := func() bool {
		if !c.phoebeStarAvailable() {
			return phoebeForteFull(c, support)
		}
		if c.phoebeConfessionReady() {
			return true
		}
		return false
	}

	if !condition() {
		return phoebeActionUnavailable
	}

	outerStart := time.Now()
	for condition() {
		if time.Since(outerStart) > 2*time.Second {
			return phoebeActionTimeout
		}
		holdStarted := time.Now()
		holdStop := func() bool {
			return !condition() && time.Since(holdStarted) >= 400*time.Millisecond
		}
		if support {
			c.holdSkillUntil(1*time.Second, 50*time.Millisecond, holdStop)
		} else {
			c.holdHeavyUntil(1*time.Second, 50*time.Millisecond, holdStop)
		}
		if c.flying() {
			waitDeadline := time.Now().Add(2 * time.Second)
			for c.flying() && time.Now().Before(waitDeadline) {
				c.attack()
				c.sleep(100 * time.Millisecond)
			}
			c.sleep(100 * time.Millisecond)
			outerStart = time.Now()
		}
		c.sleep(100 * time.Millisecond)
	}
	c.rightClickFor(50 * time.Millisecond)
	c.state.phoebeStarLatched = true
	phoebeResetAction(c, support)
	c.state.phoebeEnterStatusCount++
	return phoebeActionSuccess
}

// phoebeStarflashCombo mirrors ok-ww Phoebe.starflash_combo():
// attack loop until forte_full → right_click → perform_heavy_attack.
func phoebeStarflashCombo(c combatActor, support bool) {
	start := time.Now()
	checkpoint := start
	condition := func() bool {
		if !c.phoebeStarAvailable() {
			return c.forteFull()
		}
		if c.phoebeConfessionReady() {
			return true
		}
		return false
	}

	if !condition() && !phoebeForteFull(c, support) {
		for !phoebeForteFull(c, support) {
			if c.flying() {
				c.waitDown(2 * time.Second)
			}
			c.attack()
			if time.Since(start) > 5*time.Second {
				return
			}
			if time.Since(checkpoint) > 1*time.Second {
				if condition() || phoebeForteEmpty(c) {
					return
				}
				checkpoint = time.Now()
			}
			c.sleep(100 * time.Millisecond)
		}
		c.rightClickFor(50 * time.Millisecond)
	}

	if phoebePerformHeavyAttack(c, support) {
		c.state.phoebeStarflashCount++
	}
}

func phoebePerformHeavyAttack(c combatActor, support bool) bool {
	if phoebeAbsolutionOrConfession(c, support) != phoebeActionUnavailable {
		return false
	}
	outerStart := time.Now()
	for phoebeForteFull(c, support) {
		if time.Since(outerStart) > 2*time.Second {
			return false
		}
		c.holdHeavyUntil(500*time.Millisecond, 50*time.Millisecond, func() bool {
			return !phoebeForteFull(c, support) || c.flying()
		})
		if c.flying() {
			waitDeadline := time.Now().Add(2 * time.Second)
			for c.flying() && time.Now().Before(waitDeadline) {
				c.attack()
				c.sleep(100 * time.Millisecond)
			}
			c.sleep(100 * time.Millisecond)
			outerStart = time.Now()
			continue
		}
		if !phoebeForteFull(c, support) {
			return true
		}
		c.sleep(100 * time.Millisecond)
	}
	return true
}

func phoebeClickResonanceOnce(c combatActor) bool {
	start := time.Now()
	for phoebeResonanceAvailable(c) {
		if time.Since(start) > 500*time.Millisecond {
			return true
		}
		c.forceSkill()
		c.sleep(100 * time.Millisecond)
	}
	return false
}

func phoebeClickResonance(c combatActor, sendClick bool) bool {
	start := time.Now()
	clicked := false
	lastOp := "click"
	for phoebeResonanceAvailable(c) && time.Since(start) < 15*time.Second {
		if sendClick && lastOp == "resonance" {
			c.attack()
			lastOp = "click"
		} else if c.forceSkill() {
			clicked = true
			lastOp = "resonance"
		}
		c.sleep(100 * time.Millisecond)
	}
	return clicked
}

func phoebeClickLiberation(c combatActor, sendClick bool) bool {
	if !phoebeLiberationAvailable(c) {
		return false
	}
	start := time.Now()
	clicked := false
	for time.Since(start) < 800*time.Millisecond && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05) {
		if sendClick {
			c.attack()
		}
		c.forceLiberation()
		clicked = true
		c.sleep(100 * time.Millisecond)
	}
	if !clicked {
		retryDeadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(retryDeadline) && c.currentLiberation() > 0.001 {
			if sendClick {
				c.attack()
			}
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
		if sendClick {
			c.attack()
		}
		c.sleep(100 * time.Millisecond)
	}
	if c.isCurrentChar() || screenAnalyzer.Liberation || c.currentLiberation() > 0.05 {
		return false
	}
	freezeStart := time.Now()
	backDeadline := time.Now().Add(7 * time.Second)
	for time.Now().Before(backDeadline) && !c.isCurrentChar() {
		if sendClick {
			c.attack()
		}
		c.sleep(100 * time.Millisecond)
	}
	if !c.isCurrentChar() {
		return false
	}
	c.addFreezeDuration(time.Since(freezeStart))
	c.state.lastLiberation = time.Now()
	return true
}

func phoebeResonanceAvailable(c combatActor) bool {
	if !c.resonanceNoCD() {
		return false
	}
	return c.freezeElapsed(c.state.lastResonance, c.state.lastResonanceFreeze) >= 2*time.Second
}

func phoebeLiberationAvailable(c combatActor) bool {
	return c.param.UseLiberation && c.liberationNoCD() && (screenAnalyzer.Liberation || c.currentLiberation() > 0.05)
}

func phoebeZaniLinkage(c combatActor, support bool) bool {
	zaniState := phoebeFindZaniState(c)
	if zaniState == nil {
		return false
	}

	if screenAnalyzer.ZaniBlazesPct >= 0.9 {
		if !phoebeResonanceAvailable(c) {
			if !zaniState.zaniInLiberation || zaniLiberationTimeLeft(combatActor{action: c.action, state: zaniState}) > 3*time.Second {
				c.attackFor(1 * time.Second)
			}
		} else {
			phoebeClickResonance(c, false)
		}
		return true
	}

	if zaniState.zaniInLiberation {
		phoebeCastRemainingSkills(c, support, true)
		return true
	}
	return false
}

func phoebeFindZaniState(c combatActor) *combatCharState {
	for _, name := range []string{"zani", "zani2"} {
		if state := c.action.charStates[name]; state != nil {
			return state
		}
	}
	return nil
}

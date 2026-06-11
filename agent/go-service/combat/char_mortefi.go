package combat

import "time"

// performMortefi mirrors ok-ww Mortefi.do_perform():
//
//	wait_down → liberation → resonance → echo → !liberated→retry_liberation(wait 1s) → switch
func performMortefi(c combatActor) {
	c.waitDown(1200 * time.Millisecond)
	liberated := mortefiClickLiberation(c)
	mortefiClickResonance(c)
	c.echoWait(1 * time.Second)
	if !liberated {
		mortefiTryLiberationWait(c, 1*time.Second)
	}
	c.requestSwitch()
}

func mortefiClickResonance(c combatActor) bool {
	if c.currentResonance() <= 0.05 {
		return false
	}
	start := time.Now()
	clicked := false
	lastOp := "click"
	for c.currentResonance() > 0.05 && time.Since(start) < 15*time.Second {
		if lastOp == "resonance" {
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

func mortefiTryLiberationWait(c combatActor, wait time.Duration) bool {
	if wait <= 0 {
		return mortefiClickLiberation(c)
	}
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		if mortefiClickLiberation(c) {
			return true
		}
		c.sleep(100 * time.Millisecond)
	}
	return false
}

func mortefiClickLiberation(c combatActor) bool {
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

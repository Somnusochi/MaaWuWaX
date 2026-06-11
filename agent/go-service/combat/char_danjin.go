package combat

import "time"

// performDanjin mirrors ok-ww Danjin.do_perform():
//
//	liberation→sleep(1.2s)→echo(2s)→switch /
//	forte_full+intro→heavy(0.8s)→normal attack→switch /
//	intro→attack(1.1s) / wait_down→attack(0.4s)→resonance spam(1.1s)→switch
func performDanjin(c combatActor) {
	if danjinClickLiberation(c) {
		c.sleep(1200 * time.Millisecond)
		danjinEchoWait(c, 2*time.Second)
		c.requestSwitch()
		return
	}

	if c.forteFull() && c.recentlyIntroSwitchedIn(1800*time.Millisecond) {
		c.heavy(800 * time.Millisecond)
		c.sleep(200 * time.Millisecond)
		c.attack()
		c.sleep(100 * time.Millisecond)
		c.requestSwitch()
		return
	}
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		c.attackFor(1100 * time.Millisecond)
	} else {
		c.waitDown(900 * time.Millisecond)
		c.attackFor(400 * time.Millisecond)
	}
	// Python uses continues_click(get_resonance_key(), 1.1, interval=0.2) to send the
	// resonance key directly; Go uses Combat_RotationSkill1 which maps to the resonance
	// key through MaaFramework's task pipeline (equivalent behavior).
	start := time.Now()
	for time.Since(start) < 1100*time.Millisecond {
		c.run("Combat_RotationSkill1")
		c.sleep(200 * time.Millisecond)
	}
	c.state.lastResonance = time.Now()
	c.requestSwitch()
}

// danjinClickLiberation mirrors ok-ww Danjin.click_liberation():
// standard liberation cast with finishLiberationCast.
func danjinClickLiberation(c combatActor) bool {
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

func danjinEchoWait(c combatActor, wait time.Duration) bool {
	if wait <= 0 {
		return c.echo()
	}
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		if c.echo() {
			return true
		}
		c.sleep(100 * time.Millisecond)
	}
	return c.echo()
}

package combat

import "time"

// performQiuyuan mirrors ok-ww Qiuyuan.do_perform():
//
//	intro(1.17s attack) → flying→wait_down → loop(timeout=1.2/4s sub_dps):
//	  echo / liberation(reset timeout) / heavy_click_forte→switch /
//	  flying→shorekeeper_dodge / normal attack → resonance→switch
func performQiuyuan(c combatActor) {
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		c.attackFor(1170 * time.Millisecond)
	}
	if c.flying() {
		c.waitDown(900 * time.Millisecond)
	}
	start := time.Now()
	timeout := 1200 * time.Millisecond
	subDPSIntro := c.recentlyIntroSwitchedIn(1800*time.Millisecond) && c.switchedFromRole(roleSub, 1800*time.Millisecond)
	if subDPSIntro {
		timeout = 4 * time.Second
	}
	for time.Since(start) < timeout {
		if c.echoImmediate() {
			c.sleep(50 * time.Millisecond)
		}
		if time.Since(start) < 500*time.Millisecond && qiuyuanClickLiberation(c) {
			start = time.Now()
			if subDPSIntro {
				timeout = 4 * time.Second
			} else {
				timeout = 1200 * time.Millisecond
			}
		}
		if c.mouseForteFull() {
			c.holdHeavyUntil(1200*time.Millisecond, 50*time.Millisecond, func() bool {
				return !c.mouseForteFull()
			})
			c.sleep(70 * time.Millisecond)
			c.requestSwitch()
			return
		}
		if c.flying() && !c.mouseForteFull() {
			shorekeeperAutoDodge(c, func() bool { return c.flying() })
		}
		c.attack()
		c.sleep(80 * time.Millisecond)
	}
	qiuyuanClickResonance(c)
	c.requestSwitch()
}

func qiuyuanClickLiberation(c combatActor) bool {
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

func qiuyuanClickResonance(c combatActor) bool {
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

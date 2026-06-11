package combat

import "time"

// performDouling mirrors ok-ww Douling.do_perform():
//
//	intro(1.2s attack) → loop(1s, until con_full): liberation/resonance/attack → echo(immediate) → switch
//	Resonance uses defaultClickResonance which matches click_resonance(send_click=True, time_out=0).
func performDouling(c combatActor) {
	startTime := time.Now()
	if c.recentlyIntroSwitchedIn(1800 * time.Millisecond) {
		c.attackFor(1200 * time.Millisecond)
		startTime = time.Now()
	}
	for time.Since(startTime) < 1*time.Second && screenAnalyzer.ConcertoPct < 1.0 {
		if defaultClickLiberation(c) {
			c.sleep(1 * time.Millisecond)
			continue
		}
		if defaultClickResonance(c) {
			c.sleep(1 * time.Millisecond)
			continue
		}
		c.attack()
		c.sleep(100 * time.Millisecond)
	}
	c.echoImmediate()
	c.requestSwitch()
}

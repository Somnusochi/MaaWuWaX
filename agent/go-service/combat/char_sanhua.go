package combat

import "time"

// performSanhua mirrors ok-ww Sanhua.do_perform():
//
//	mouse_down → wait_down → liberation(send_click=False) / resonance → sleep time calc →
//	mouse_up → sleep(0.8s) → liber: resonance+0.3s → con_full: echo → switch
func performSanhua(c combatActor) {
	ctrl := c.ctx.GetTasker().GetController()
	c.sleep(20 * time.Millisecond)
	start := time.Now()
	startFreeze := screenAnalyzer.FreezeDuration
	ctrl.PostTouchDown(0, 640, 360, 1).Wait()
	sanhuaWaitDownNoClick(c, 900*time.Millisecond)
	liberClicked := sanhuaClickLiberation(c)
	if !liberClicked && c.currentResonance() > 0.05 {
		ctrl.PostTouchUp(0).Wait()
		sanhuaClickResonance(c)
		start = time.Now()
		startFreeze = screenAnalyzer.FreezeDuration
		ctrl.PostTouchDown(0, 640, 360, 1).Wait()
		c.sleep(100 * time.Millisecond)
	}
	sleepTime := 850 * time.Millisecond
	if liberClicked {
		sleepTime += 100 * time.Millisecond
		c.sleep(150 * time.Millisecond)
	}
	sleepTime -= c.freezeElapsed(start, startFreeze)
	if sleepTime < 0 {
		sleepTime = 0
	}
	c.sleep(sleepTime)
	ctrl.PostTouchUp(0).Wait()
	c.sleep(800 * time.Millisecond)
	if liberClicked {
		sanhuaClickResonance(c)
		c.sleep(300 * time.Millisecond)
	}
	if screenAnalyzer.ConcertoPct >= 1.0 {
		c.echoWait(1 * time.Second)
	}
	c.requestSwitch()
}

func sanhuaWaitDownNoClick(c combatActor, timeout time.Duration) bool {
	if timeout <= 0 {
		timeout = 2500 * time.Millisecond
	}
	if !c.flying() {
		return true
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !c.flying() {
			return true
		}
		c.sleep(100 * time.Millisecond)
	}
	return !c.flying()
}

func sanhuaClickLiberation(c combatActor) bool {
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

func sanhuaClickResonance(c combatActor) bool {
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

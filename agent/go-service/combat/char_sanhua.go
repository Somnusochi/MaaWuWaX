package combat

import "time"

func performSanhua(c combatActor) {
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostTouchDown(0, 640, 360, 1).Wait()
	c.sleep(420 * time.Millisecond)
	liberClicked := c.liberation()
	if !liberClicked {
		c.skill()
	}
	c.sleep(350 * time.Millisecond)
	ctrl.PostTouchUp(0).Wait()
	c.sleep(800 * time.Millisecond)
	if liberClicked {
		c.skill()
		c.sleep(300 * time.Millisecond)
	}
	c.echo()
	c.requestSwitch()
}

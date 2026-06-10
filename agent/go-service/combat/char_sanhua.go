package combat
import "time"
func performSanhua(c combatActor) {
	ctrl := c.ctx.GetTasker().GetController()
	ctrl.PostTouchDown(0,640,360,1).Wait(); c.sleep(420*time.Millisecond)
	if !c.liberation() { c.skill() }; c.sleep(350*time.Millisecond)
	ctrl.PostTouchUp(0).Wait(); c.sleep(250*time.Millisecond)
	c.echo(); c.requestSwitch()
}

// Package mouse provides mouse control utilities for Wuthering Waves on macOS.
package mouse

import (
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
)

// MiddleClick performs a middle-button click (contact=2 in MaaFramework convention).
// Used in WuWa to lock camera onto the target under crosshair.
func MiddleClick(ctrl *maa.Controller) {
	ctrl.PostClickV2(640, 360, 2, 1).Wait()
	time.Sleep(50 * time.Millisecond)
}

// Scroll performs a scroll action. Positive dy scrolls up, negative dy scrolls down.
func Scroll(ctrl *maa.Controller, dx, dy int32) {
	ctrl.PostScroll(dx, dy).Wait()
	time.Sleep(50 * time.Millisecond)
}

// ScrollUp scrolls up by the given amount.
func ScrollUp(ctrl *maa.Controller, amount int32) {
	Scroll(ctrl, 0, amount)
}

// ScrollDown scrolls down by the given amount.
func ScrollDown(ctrl *maa.Controller, amount int32) {
	Scroll(ctrl, 0, -amount)
}

// ClickCenter clicks the screen center (640, 360 at 720p).
func ClickCenter(ctrl *maa.Controller) {
	ctrl.PostClick(640, 360).Wait()
	time.Sleep(50 * time.Millisecond)
}

// ClickAt clicks at the specified coordinates.
func ClickAt(ctrl *maa.Controller, x, y int32) {
	ctrl.PostClick(x, y).Wait()
	time.Sleep(50 * time.Millisecond)
}

// Drag performs a drag from (x1,y1) to (x2,y2) over the given duration.
func Drag(ctrl *maa.Controller, x1, y1, x2, y2 int32, duration time.Duration) {
	ctrl.PostSwipe(x1, y1, x2, y2, duration).Wait()
	time.Sleep(50 * time.Millisecond)
}

// RotateCamera rotates the camera by simulating a mouse drag from center.
// dx, dy are the relative offsets from center.
func RotateCamera(ctrl *maa.Controller, dx, dy int32) {
	cx, cy := int32(640), int32(360)
	ctrl.PostSwipe(cx, cy, cx+dx, cy+dy, 100*time.Millisecond).Wait()
	time.Sleep(50 * time.Millisecond)
}

// ResetToCenter moves the cursor back to screen center using ALT+click trick.
// On macOS this releases mouse capture and re-centers.
func ResetToCenter(ctrl *maa.Controller) {
	// Hold ALT to release mouse, click center, release ALT.
	const altCode int32 = 58 // CGKeyCode for ALT
	ctrl.PostKeyDown(altCode).Wait()
	time.Sleep(30 * time.Millisecond)
	ctrl.PostClick(640, 360).Wait()
	time.Sleep(30 * time.Millisecond)
	ctrl.PostKeyUp(altCode).Wait()
	time.Sleep(50 * time.Millisecond)
}

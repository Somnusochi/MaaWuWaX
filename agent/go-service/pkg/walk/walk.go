// Package walk provides WASD direction walking utilities for Wuthering Waves.
// Uses macOS CGKeyCode via the keycode package.
package walk

import (
	"time"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/keycode"
	maa "github.com/MaaXYZ/maa-framework-go/v4"
	"github.com/rs/zerolog/log"
)

// Direction represents a WASD movement direction.
type Direction int

const (
	Forward Direction = iota
	Back
	Left
	Right
)

// keyCodes maps directions to their CGKeyCode.
var keyCodes = map[Direction]int32{
	Forward: keycode.MustCode("W"),
	Back:    keycode.MustCode("S"),
	Left:    keycode.MustCode("A"),
	Right:   keycode.MustCode("D"),
}

// Walk presses the given direction key, holds for the specified duration, then releases.
func Walk(ctrl *maa.Controller, dir Direction, hold time.Duration) {
	code := keyCodes[dir]
	ctrl.PostKeyDown(code).Wait()
	time.Sleep(hold)
	ctrl.PostKeyUp(code).Wait()
	time.Sleep(50 * time.Millisecond)
}

// Sprint holds Shift while walking in the given direction.
func Sprint(ctrl *maa.Controller, dir Direction, hold time.Duration) {
	dirCode := keyCodes[dir]
	shiftCode := keycode.MustCode("SHIFT")
	ctrl.PostKeyDown(shiftCode).Wait()
	time.Sleep(30 * time.Millisecond)
	ctrl.PostKeyDown(dirCode).Wait()
	time.Sleep(hold)
	ctrl.PostKeyUp(dirCode).Wait()
	ctrl.PostKeyUp(shiftCode).Wait()
	time.Sleep(50 * time.Millisecond)
}

// StepForward sends a quick W press (useful for small adjustments).
func StepForward(ctrl *maa.Controller) {
	Walk(ctrl, Forward, 200*time.Millisecond)
}

// RunForward runs forward for the given duration using default run speed.
func RunForward(ctrl *maa.Controller, hold time.Duration) {
	Walk(ctrl, Forward, hold)
}

// WalkTo walks in a direction for a specified number of steps, each step being stepMs long.
func WalkTo(ctrl *maa.Controller, dir Direction, steps int, stepMs int) {
	code := keyCodes[dir]
	for i := 0; i < steps; i++ {
		ctrl.PostKeyDown(code).Wait()
		time.Sleep(time.Duration(stepMs) * time.Millisecond)
		ctrl.PostKeyUp(code).Wait()
		time.Sleep(30 * time.Millisecond)
	}
	log.Debug().
		Str("component", "Walk").
		Int("steps", steps).
		Int("stepMs", stepMs).
		Msg("walk complete")
}

// StopAll releases all WASD keys to ensure the character stops.
func StopAll(ctrl *maa.Controller) {
	for _, code := range keyCodes {
		ctrl.PostKeyUp(code).Wait()
	}
	time.Sleep(30 * time.Millisecond)
}

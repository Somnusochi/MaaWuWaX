// Package parentwatch monitors the parent process and exits when it dies.
package parentwatch

import (
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

const PollInterval = 1 * time.Second
const logComponent = "parent-watcher"

// IsParentAlive returns true if the stored parent process is still running.
// This is the exported wrapper for use by TaskerSink and other components.
var parentPID int

func init() {
	parentPID = os.Getppid()
}

// IsParentAlive checks if the parent process is still alive.
func IsParentAlive() bool {
	if parentPID <= 1 {
		return true // no valid parent, assume alive
	}
	return isProcessAlive(parentPID)
}

// Start begins monitoring the parent process in a background goroutine.
func Start() {
	if parentPID <= 1 {
		log.Warn().
			Str("component", logComponent).
			Int("parent_pid", parentPID).
			Msg("invalid parent pid, watcher disabled")
		return
	}

	log.Info().
		Str("component", logComponent).
		Int("parent_pid", parentPID).
		Msg("parent process watcher started")

	go func() {
		ticker := time.NewTicker(PollInterval)
		defer ticker.Stop()

		for range ticker.C {
			if !isProcessAlive(parentPID) {
				log.Warn().
					Str("component", logComponent).
					Int("parent_pid", parentPID).
					Msg("parent process has exited; shutting down")
				os.Exit(0)
			}
		}
	}()
}

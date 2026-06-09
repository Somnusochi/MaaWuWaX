// Package processcheck provides a supplementary process health check
// that monitors the parent process and logs status changes.
// The primary parent monitoring is handled by pkg/parentwatch.
package processcheck

import (
	"time"

	"github.com/rs/zerolog/log"

	"github.com/MaaWuWaX/MaaWuWaX/agent/go-service/pkg/parentwatch"
)

// StartPeriodicCheck begins a periodic health check loop.
// It logs warnings when the parent process is no longer alive.
// This supplements the parentwatch.Start() which calls os.Exit directly.
func StartPeriodicCheck(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if !parentwatch.IsParentAlive() {
				log.Warn().
					Str("component", "ProcessCheck").
					Msg("parent process no longer alive (supplementary check)")
				return
			}
		}
	}()
}

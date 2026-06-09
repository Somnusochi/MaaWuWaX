// Package maafocus provides structured logging utilities for MaaFramework tasks.
// It prints formatted focus messages to help users track task progress.
package maafocus

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// Print logs a focus message indicating a notable event in a task.
func Print(component, message string) {
	log.Info().
		Str("component", component).
		Str("focus", "true").
		Msg(message)
}

// Printf logs a formatted focus message.
func Printf(component, format string, args ...any) {
	Print(component, fmt.Sprintf(format, args...))
}

// PrintWithDetail logs a focus message with additional key-value detail.
func PrintWithDetail(component, message string, detail map[string]any) {
	e := log.Info().
		Str("component", component).
		Str("focus", "true")
	for k, v := range detail {
		e = e.Interface(k, v)
	}
	e.Msg(message)
}

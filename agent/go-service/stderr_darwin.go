//go:build darwin

package main

import (
	"os"
	"syscall"
)

// redirectStderr redirects stderr to a file on macOS.
// MaaFramework may print diagnostic info to stderr; capturing it helps debugging.
func redirectStderr(path string) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	_ = syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
}

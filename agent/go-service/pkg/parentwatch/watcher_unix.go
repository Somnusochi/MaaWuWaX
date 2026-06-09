//go:build !windows

package parentwatch

import "syscall"

// isProcessAlive checks if a process with the given PID is still running.
// On POSIX systems, syscall.Kill(pid, 0) returns ESRCH if the process does not exist.
func isProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}

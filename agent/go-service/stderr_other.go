//go:build !darwin

package main

// redirectStderr is a no-op on non-macOS platforms.
func redirectStderr(path string) {
	_ = path
}

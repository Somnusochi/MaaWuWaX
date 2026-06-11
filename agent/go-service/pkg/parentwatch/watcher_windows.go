//go:build windows

package parentwatch

import "golang.org/x/sys/windows"

// isProcessAlive checks whether the process handle is still unsignaled.
func isProcessAlive(pid int) bool {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	state, err := windows.WaitForSingleObject(handle, 0)
	if err != nil {
		return false
	}
	return state != uint32(windows.WAIT_OBJECT_0)
}

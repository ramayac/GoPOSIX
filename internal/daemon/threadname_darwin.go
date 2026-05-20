//go:build darwin

package daemon

// setWorkerThreadName is a no-op on macOS (PR_SET_NAME is Linux-only).
func setWorkerThreadName(id int) {}

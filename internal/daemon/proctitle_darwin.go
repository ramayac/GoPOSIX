//go:build darwin

package daemon

// setProcTitle is a no-op on macOS (argv overwrite is Linux-specific).
func setProcTitle(title string) {}

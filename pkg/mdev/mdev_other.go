//go:build !linux

package mdev

import "fmt"

// createDevNode is not supported on non-Linux platforms.
func createDevNode(d DevNode) error {
	return fmt.Errorf("mdev: not supported on this platform")
}

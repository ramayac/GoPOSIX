//go:build linux

package mdev

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// createDevNode creates a device node in /dev using mknod(2).
func createDevNode(d DevNode) error {
	path := filepath.Join("/dev", d.Name)

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Remove existing node (ignore error)
	os.Remove(path)

	var mode uint32 = 0660
	var devType uint32
	if d.Type == "block" {
		devType = unix.S_IFBLK
	} else {
		devType = unix.S_IFCHR
	}

	dev := unix.Mkdev(uint32(d.Major), uint32(d.Minor))
	if err := unix.Mknod(path, devType|mode, int(dev)); err != nil {
		return fmt.Errorf("mknod %s: %v", path, err)
	}
	return nil
}

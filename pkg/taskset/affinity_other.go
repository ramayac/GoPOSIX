//go:build !linux

package taskset

import (
	"fmt"
)

func getAffinity(pid int) (string, error) {
	if pid == 0 {
		return "", fmt.Errorf("invalid pid: %d", pid)
	}
	return "1", nil
}

func setAffinity(pid int, maskStr string) (string, string, error) {
	if pid == 0 {
		return "", "", fmt.Errorf("invalid pid: %d", pid)
	}
	return "1", "1", nil
}

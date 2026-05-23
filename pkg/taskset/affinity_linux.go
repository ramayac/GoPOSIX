//go:build linux

package taskset

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func getAffinity(pid int) (string, error) {
	var set unix.CPUSet
	err := unix.SchedGetaffinity(pid, &set)
	if err != nil {
		return "", err
	}
	var val uint64
	for cpu := 0; cpu < 64; cpu++ {
		if set.IsSet(cpu) {
			val |= (1 << cpu)
		}
	}
	return fmt.Sprintf("%x", val), nil
}

func setAffinity(pid int, maskStr string) (string, string, error) {
	cleanMask := strings.TrimPrefix(maskStr, "0x")
	cleanMask = strings.TrimPrefix(cleanMask, "0X")
	val, err := strconv.ParseUint(cleanMask, 16, 64)
	if err != nil {
		return "", "", fmt.Errorf("invalid hex mask: %s", maskStr)
	}

	var oldSet unix.CPUSet
	var oldMask string
	if errGet := unix.SchedGetaffinity(pid, &oldSet); errGet == nil {
		var oldVal uint64
		for cpu := 0; cpu < 64; cpu++ {
			if oldSet.IsSet(cpu) {
				oldVal |= (1 << cpu)
			}
		}
		oldMask = fmt.Sprintf("%x", oldVal)
	} else {
		oldMask = "1"
	}

	var newSet unix.CPUSet
	newSet.Zero()
	for cpu := 0; cpu < 64; cpu++ {
		if (val & (1 << cpu)) != 0 {
			newSet.Set(cpu)
		}
	}

	err = unix.SchedSetaffinity(pid, &newSet)
	if err != nil {
		return "", "", err
	}

	return oldMask, fmt.Sprintf("%x", val), nil
}

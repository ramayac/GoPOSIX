// Package common provides shared helpers for goposix utilities.
package common

import (
	"os/user"
	"strconv"
)

// LookupUID resolves a user identifier (name or numeric) to a UID.
// It returns -1 when the user cannot be found.
func LookupUID(name string) int {
	if val, err := strconv.Atoi(name); err == nil {
		return val
	}
	if u, err := user.Lookup(name); err == nil {
		if val, err := strconv.Atoi(u.Uid); err == nil {
			return val
		}
	}
	return -1
}

// LookupGID resolves a group identifier (name or numeric) to a GID.
// It returns -1 when the group cannot be found.
func LookupGID(name string) int {
	if val, err := strconv.Atoi(name); err == nil {
		return val
	}
	if g, err := user.LookupGroup(name); err == nil {
		if val, err := strconv.Atoi(g.Gid); err == nil {
			return val
		}
	}
	return -1
}

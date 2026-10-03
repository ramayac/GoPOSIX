//go:build linux

package mount

import (
	"bytes"
	"testing"
)

func TestDoMountFailsWithoutPrivileges(t *testing.T) {
	// Mounting requires CAP_SYS_ADMIN; as a regular user (and in CI) the
	// syscall fails. Verify the error path renders for both output modes.
	for _, jsonMode := range []bool{false, true} {
		var out, errBuf bytes.Buffer
		code := doMount("nodev", "/nonexistent-mountpoint", "ext4", "ro", jsonMode, &out, &errBuf)
		if code != 1 {
			t.Errorf("jsonMode=%v: expected exit 1, got %d", jsonMode, code)
		}
		if !bytes.Contains(errBuf.Bytes(), []byte("mount:")) {
			t.Errorf("jsonMode=%v: stderr = %q", jsonMode, errBuf.String())
		}
	}
}

package chgrp

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestChgrpMissingArgs(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{}, nil, &out, &out, "")
	if rc != 1 {
		t.Errorf("expected 1, got %d", rc)
	}
}

func TestChgrpJSON(t *testing.T) {
	var out bytes.Buffer
	f, _ := os.CreateTemp("", "chgrp")
	defer os.Remove(f.Name())

	rc := run([]string{"--json", "0", f.Name()}, nil, &out, &out, "")
	// Might fail if not root, so we just check it runs and outputs json
	_ = rc
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}

func TestLookupGIDNumeric(t *testing.T) {
	if got := lookupGID("12345"); got != 12345 {
		t.Errorf("lookupGID(\"12345\") = %d, want 12345", got)
	}
}

func TestLookupGIDByName(t *testing.T) {
	// The root group exists on every POSIX system.
	if got := lookupGID("root"); got != 0 {
		t.Errorf("lookupGID(\"root\") = %d, want 0", got)
	}
}

func TestLookupGIDUnknown(t *testing.T) {
	if got := lookupGID("no-such-group-xyz"); got != -1 {
		t.Errorf("lookupGID(unknown) = %d, want -1", got)
	}
}

func TestChgrpNonexistentFile(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"0", "/nonexistent-xyz-file"}, nil, &out, &errBuf, ""); code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

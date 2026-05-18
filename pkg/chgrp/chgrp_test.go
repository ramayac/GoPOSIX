package chgrp

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestChgrpMissingArgs(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{}, &out)
	if rc != 1 {
		t.Errorf("expected 1, got %d", rc)
	}
}

func TestChgrpJSON(t *testing.T) {
	var out bytes.Buffer
	f, _ := os.CreateTemp("", "chgrp")
	defer os.Remove(f.Name())

	rc := run([]string{"--json", "0", f.Name()}, &out)
	// Might fail if not root, so we just check it runs and outputs json
	_ = rc
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}

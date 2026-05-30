package id

import (
	"bytes"
	"strings"
	"testing"
)

func TestIdRun(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "uid=") {
		t.Error("expected output uid=")
	}
}

func TestIdJSON(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--json"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("expected 0, got %d", rc)
	}
	if !strings.Contains(out.String(), "command") {
		t.Errorf("expected JSON, got %s", out.String())
	}
}

func TestIdUserFlag(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-u"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("id -u: exit %d", rc)
	}
	outStr := strings.TrimSpace(out.String())
	if outStr == "" {
		t.Error("id -u produced no output")
	}
}

func TestIdGroupFlag(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-g"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("id -g: exit %d", rc)
	}
	outStr := strings.TrimSpace(out.String())
	if outStr == "" {
		t.Error("id -g produced no output")
	}
}

func TestIdGroupsFlag(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-G"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("id -G: exit %d", rc)
	}
	outStr := strings.TrimSpace(out.String())
	if outStr == "" {
		t.Error("id -G produced no output")
	}
}

func TestIdNameFlag(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-n"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("id -n: exit %d", rc)
	}
}

func TestIdUserWithNameFlag(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-un"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("id -un: exit %d", rc)
	}
	outStr := strings.TrimSpace(out.String())
	if outStr == "" {
		t.Error("id -un produced no output")
	}
}

func TestIdGroupWithNameFlag(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-gn"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("id -gn: exit %d", rc)
	}
	outStr := strings.TrimSpace(out.String())
	if outStr == "" {
		t.Error("id -gn produced no output")
	}
}

func TestIdGroupsWithNameFlag(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"-Gn"}, nil, &out, &out, "")
	if rc != 0 {
		t.Errorf("id -Gn: exit %d", rc)
	}
	outStr := strings.TrimSpace(out.String())
	if outStr == "" {
		t.Error("id -Gn produced no output")
	}
}

func TestIdBadFlag(t *testing.T) {
	var out bytes.Buffer
	rc := run([]string{"--no-such-flag"}, nil, &out, &out, "")
	if rc != 1 {
		t.Errorf("id --no-such-flag: exit %d, want 1", rc)
	}
}

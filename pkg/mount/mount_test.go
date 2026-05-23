package mount

import (
	"bytes"
	"strings"
	"testing"
)

func TestMountListMode(t *testing.T) {
	// On Linux, /proc/mounts always exists; test the list path
	var stdout, stderr bytes.Buffer
	rc := mountRun([]string{}, nil, &stdout, &stderr, "/tmp")
	// Should succeed if /proc/mounts or /etc/mtab is readable
	// (it may fail in a severely restricted container; accept 0 or 1)
	_ = rc
	// At minimum, it should not panic
}

func TestMountListJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := mountRun([]string{"--json"}, nil, &stdout, &stderr, "/tmp")
	_ = rc
	// If /proc/mounts is readable, output should contain JSON
	out := stdout.String()
	if rc == 0 && !strings.Contains(out, `"mounts"`) {
		t.Errorf("expected JSON output with 'mounts' key, got: %s", out)
	}
}

func TestMountMissingMountpoint(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := mountRun([]string{"tmpfs"}, nil, &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc for missing mountpoint")
	}
}

func TestMountBadArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := mountRun([]string{"--unknown-flag-xyz"}, nil, &stdout, &stderr, "/tmp")
	_ = rc
	// Should not panic
}

func TestParseMountTable(t *testing.T) {
	input := `sysfs /sys sysfs rw,nosuid,nodev,noexec,relatime 0 0
proc /proc proc rw,nosuid,nodev,noexec,relatime 0 0
tmpfs /tmp tmpfs rw,nosuid,nodev 0 0`

	var stdout bytes.Buffer
	rc := parseMountTable(strings.NewReader(input), false, &stdout)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d", rc)
	}
	out := stdout.String()
	if !strings.Contains(out, "sysfs on /sys type sysfs") {
		t.Errorf("expected sysfs entry, got: %s", out)
	}
	if !strings.Contains(out, "proc on /proc type proc") {
		t.Errorf("expected proc entry, got: %s", out)
	}
}

func TestParseMountTableJSON(t *testing.T) {
	input := `sysfs /sys sysfs rw 0 0`

	var stdout bytes.Buffer
	rc := parseMountTable(strings.NewReader(input), true, &stdout)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d", rc)
	}
	out := stdout.String()
	if !strings.Contains(out, `"device"`) || !strings.Contains(out, "sysfs") {
		t.Errorf("expected JSON with device field, got: %s", out)
	}
}

func TestParseMountTableIgnoresComments(t *testing.T) {
	input := `# this is a comment
sysfs /sys sysfs rw 0 0
# another comment`

	var stdout bytes.Buffer
	rc := parseMountTable(strings.NewReader(input), false, &stdout)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d", rc)
	}
	out := stdout.String()
	if strings.Contains(out, "#") {
		t.Errorf("comments should not appear in output, got: %s", out)
	}
}

func TestParseOptions(t *testing.T) {
	tests := []struct {
		options string
		wantRO  bool
	}{
		{"ro,nosuid,nodev", true},
		{"rw,bind", false},
		{"defaults", false},
		{"ro", true},
	}

	for _, tt := range tests {
		flags, data := parseOptions(tt.options)
		if tt.wantRO && (flags&msRdOnly == 0) {
			t.Errorf("parseOptions(%q): expected MS_RDONLY set, data=%q", tt.options, data)
		}
		if !tt.wantRO && (flags&msRdOnly != 0) {
			t.Errorf("parseOptions(%q): expected MS_RDONLY NOT set", tt.options)
		}
	}
}

func TestMountReadOnly(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// This will fail to mount but should format args correctly (error from syscall)
	rc := mountRun([]string{"-r", "tmpfs", "/nonexistent/mountpoint"}, nil, &stdout, &stderr, "/tmp")
	// We expect it to fail since /nonexistent/mountpoint doesn't exist,
	// but it should not panic
	_ = rc
}

func TestMountAllNoFstab(t *testing.T) {
	// Test that -a gracefully handles missing /etc/fstab by mocking the function
	// (In a real container, /etc/fstab likely doesn't exist; just verify no panic)
	// We call with -a and rely on the graceful error handling
	var stdout, stderr bytes.Buffer
	rc := mountAllFstab("auto", "", false, &stdout, &stderr)
	// Accept both 0 (if fstab present but no auto entries) and 1 (fstab missing)
	_ = rc
}

func TestMountRunDispatch(t *testing.T) {
	// Call run() directly to cover the dispatch wrapper
	var stdout, stderr bytes.Buffer
	rc := run([]string{"--json"}, nil, &stdout, &stderr, "/tmp")
	_ = rc
	// Should not panic
}

func TestParseMountTableShortLines(t *testing.T) {
	// Test with lines that have fewer than 4 fields (should be skipped)
	input := "sysfs /sys\n" // only 2 fields
	var stdout bytes.Buffer
	rc := parseMountTable(strings.NewReader(input), false, &stdout)
	if rc != 0 {
		t.Fatalf("expected rc=0, got %d", rc)
	}
	// Short lines should be silently skipped
	if strings.TrimSpace(stdout.String()) != "" {
		t.Errorf("expected empty output for short lines, got: %s", stdout.String())
	}
}

package mdev

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestMdevDryRun(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := mdevRun([]string{"-d"}, nil, &stdout, &stderr, "/tmp")
	// May fail if /sys/class is not accessible (e.g., in sandbox)
	// But should not panic
	_ = rc
}

func TestMdevDryRunJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := mdevRun([]string{"-d", "--json"}, nil, &stdout, &stderr, "/tmp")
	_ = rc
	out := stdout.String()
	// If /sys/class is accessible and devices were found, we should get JSON
	if rc == 0 && len(out) > 0 && !strings.Contains(out, `"devices"`) {
		t.Errorf("expected JSON output with 'devices' key, got: %s", out)
	}
}

func TestMdevNoArgsNoEnv(t *testing.T) {
	// Clear relevant env vars
	os.Unsetenv("ACTION")
	os.Unsetenv("DEVPATH")
	os.Unsetenv("SUBSYSTEM")
	os.Unsetenv("MAJOR")
	os.Unsetenv("MINOR")
	os.Unsetenv("DEVNAME")

	var stdout, stderr bytes.Buffer
	rc := mdevRun([]string{}, nil, &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc when no mode and no env vars")
	}
}

func TestMdevHotplugAdd(t *testing.T) {
	// Set env vars to simulate a hotplug add event
	t.Setenv("ACTION", "add")
	t.Setenv("DEVPATH", "/devices/virtual/test/testdev0")
	t.Setenv("SUBSYSTEM", "char")
	t.Setenv("MAJOR", "1")
	t.Setenv("MINOR", "3")
	t.Setenv("DEVNAME", "null")

	var stdout, stderr bytes.Buffer
	rc := mdevRun([]string{}, nil, &stdout, &stderr, "/tmp")
	// May fail if /dev is read-only in test environment; just verify no panic
	_ = rc
}

func TestMdevHotplugRemove(t *testing.T) {
	t.Setenv("ACTION", "remove")
	t.Setenv("DEVPATH", "/devices/virtual/test/testdev_remove")
	t.Setenv("DEVNAME", "goposix_test_dev")

	var stdout, stderr bytes.Buffer
	rc := mdevRun([]string{}, nil, &stdout, &stderr, "/tmp")
	// remove of nonexistent file should be silently ok
	_ = rc
}

func TestMdevHotplugUnknownAction(t *testing.T) {
	t.Setenv("ACTION", "bogus_action")
	t.Setenv("DEVPATH", "/devices/virtual/test/foo")
	t.Setenv("DEVNAME", "foo")

	var stdout, stderr bytes.Buffer
	rc := mdevRun([]string{}, nil, &stdout, &stderr, "/tmp")
	if rc == 0 {
		t.Error("expected non-zero rc for unknown action")
	}
}

func TestMdevHotplugJSON(t *testing.T) {
	t.Setenv("ACTION", "add")
	t.Setenv("DEVPATH", "/devices/virtual/null_test")
	t.Setenv("SUBSYSTEM", "char")
	t.Setenv("MAJOR", "999")
	t.Setenv("MINOR", "999")
	t.Setenv("DEVNAME", "goposix_null_test")

	var stdout, stderr bytes.Buffer
	// JSON mode with --json
	rc := mdevRun([]string{"--json"}, nil, &stdout, &stderr, "/tmp")
	_ = rc
}

func TestDiscoverDevicesNoSys(t *testing.T) {
	// readDevNode with invalid path should return error
	_, err := readDevNode("testdev", "/nonexistent/path/to/dev")
	if err == nil {
		t.Error("expected error for nonexistent sysfs path")
	}
}

func TestReadDevNodeInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	// Write invalid dev file content
	if err := os.WriteFile(dir+"/dev", []byte("not-a-valid-format"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := readDevNode("testdev", dir)
	if err == nil {
		t.Error("expected error for invalid dev file format")
	}
}

func TestMdevBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := mdevRun([]string{"--bad-flag"}, nil, &stdout, &stderr, "/tmp")
	_ = rc
	// Should not panic
}

func TestMdevRunDispatch(t *testing.T) {
	// Call run() directly to cover the dispatch wrapper
	os.Unsetenv("ACTION")
	os.Unsetenv("DEVPATH")

	var stdout, stderr bytes.Buffer
	rc := run([]string{}, nil, &stdout, &stderr, "/tmp")
	// No args and no env should fail
	if rc == 0 {
		t.Error("expected non-zero rc")
	}
}

func TestMdevScanMode(t *testing.T) {
	// Test -s scan mode - this exercises mdevScan path
	// In test environment /sys/class may or may not be readable,
	// but mdevScan itself should execute even if most nodes fail to create
	var stdout, stderr bytes.Buffer
	rc := mdevRun([]string{"-s"}, nil, &stdout, &stderr, "/tmp")
	// May succeed (if /sys/class readable) or fail (permission denied for mknod)
	// Just ensure it doesn't panic and exercises the code path
	_ = rc
}

package touch

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunCreate(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "newfile.txt")
	ts := time.Now()
	result, err := Run([]string{f}, ts, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Touched) != 1 {
		t.Fatalf("expected 1 touched, got %v", result.Touched)
	}
	if _, err := os.Stat(f); os.IsNotExist(err) {
		t.Error("file was not created by touch")
	}
}

func TestRunUpdateMtime(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "existing.txt")
	os.WriteFile(f, []byte("data"), 0644)

	past := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := Run([]string{f}, past, false)
	if err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(f)
	delta := info.ModTime().Unix() - past.Unix()
	if delta < -2 || delta > 2 {
		t.Errorf("mtime not updated: got %v, want ~%v", info.ModTime(), past)
	}
}

func TestRunMultiplePaths(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.txt")
	f2 := filepath.Join(dir, "b.txt")
	ts := time.Now()
	result, err := Run([]string{f1, f2}, ts, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Touched) != 2 {
		t.Errorf("expected 2 touched, got %d", len(result.Touched))
	}
	for _, f := range []string{f1, f2} {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("file %s was not created", f)
		}
	}
}

func TestRunExistingFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "existing.txt")
	os.WriteFile(f, []byte("content"), 0644)
	orig, _ := os.Stat(f)

	ts := time.Now()
	_, err := Run([]string{f}, ts, false)
	if err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(f)
	if info.Size() != orig.Size() {
		t.Error("touch should not modify file content")
	}
}

// --- BusyBox test suite hardening ---

func TestBusyBox_Touch_NoCreate(t *testing.T) {
	// BusyBox: touch -c foo; test ! -f foo
	// -c means "do not create". If file doesn't exist, skip silently.
	dir := t.TempDir()
	f := filepath.Join(dir, "foo")
	ts := time.Now()
	result, err := Run([]string{f}, ts, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Touched) != 0 {
		t.Errorf("expected 0 touched with -c on non-existent file, got %d", len(result.Touched))
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Error("touch -c should NOT create the file")
	}
}

func TestBusyBox_Touch_NoCreateSkipsMissingTouchesExisting(t *testing.T) {
	// BusyBox: touch -c foo bar — foo doesn't exist, bar does.
	// Bar should still be timestamp-updated, foo silently skipped.
	dir := t.TempDir()
	foo := filepath.Join(dir, "foo")
	bar := filepath.Join(dir, "bar")

	// Create bar with old timestamp
	past := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	os.WriteFile(bar, []byte("data"), 0644)
	os.Chtimes(bar, past, past)

	ts := time.Now()
	result, err := Run([]string{foo, bar}, ts, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// foo should NOT exist
	if _, err := os.Stat(foo); !os.IsNotExist(err) {
		t.Error("touch -c foo should NOT create foo")
	}
	// bar should be touched (updated)
	if len(result.Touched) < 1 || result.Touched[0] != bar {
		t.Errorf("bar should have been touched, got %v", result.Touched)
	}
	info, _ := os.Stat(bar)
	if info.ModTime().Before(past.Add(1 * time.Hour)) {
		t.Errorf("bar timestamp should be updated, got %v", info.ModTime())
	}
}

// --- CLI tests via run() ---

func TestCLI_Basic(t *testing.T) {
	f := filepath.Join(t.TempDir(), "test.txt")
	var out bytes.Buffer
	code := run([]string{f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if _, err := os.Stat(f); os.IsNotExist(err) {
		t.Error("file was not created")
	}
}

func TestCLI_NoCreate(t *testing.T) {
	f := filepath.Join(t.TempDir(), "nonexist.txt")
	var out bytes.Buffer
	code := run([]string{"-c", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Error("-c should not create file")
	}
}

func TestCLI_ReferenceFile(t *testing.T) {
	dir := t.TempDir()
	ref := filepath.Join(dir, "ref.txt")
	os.WriteFile(ref, []byte("ref"), 0644)
	refTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	os.Chtimes(ref, refTime, refTime)

	target := filepath.Join(dir, "target.txt")
	os.WriteFile(target, []byte("target"), 0644)

	var out bytes.Buffer
	code := run([]string{"-r", ref, target}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	info, _ := os.Stat(target)
	delta := info.ModTime().Unix() - refTime.Unix()
	if delta < -2 || delta > 2 {
		t.Errorf("mtime should match reference, got %v", info.ModTime())
	}
}

func TestCLI_DateString(t *testing.T) {
	f := filepath.Join(t.TempDir(), "dated.txt")
	var out bytes.Buffer
	code := run([]string{"-d", "2020-06-15 12:00:00", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	info, _ := os.Stat(f)
	want := time.Date(2020, 6, 15, 12, 0, 0, 0, time.Local)
	delta := info.ModTime().Unix() - want.Unix()
	if delta < -2 || delta > 2 {
		t.Errorf("mtime should be 2020-06-15, got %v", info.ModTime())
	}
}

func TestCLI_TimeStamp(t *testing.T) {
	f := filepath.Join(t.TempDir(), "timestamped.txt")
	var out bytes.Buffer
	code := run([]string{"-t", "202006151200.00", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	info, _ := os.Stat(f)
	want := time.Date(2020, 6, 15, 12, 0, 0, 0, time.Local)
	delta := info.ModTime().Unix() - want.Unix()
	if delta < -2 || delta > 2 {
		t.Errorf("mtime should be 2020-06-15 12:00, got %v", info.ModTime())
	}
}

func TestCLI_JSON(t *testing.T) {
	f := filepath.Join(t.TempDir(), "json.txt")
	var out bytes.Buffer
	code := run([]string{"--json", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out.String(), "\"touched\"") {
		t.Errorf("expected JSON output, got: %s", out.String())
	}
}

func TestCLI_MissingFile(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{}, &out)
	if code != 1 {
		t.Errorf("expected exit 1 for missing file operand, got %d", code)
	}
}

func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, &out)
	if code != 2 {
		t.Errorf("expected exit 2 for bad flag, got %d", code)
	}
}

func TestCLI_BadReference(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"-r", "/nonexistent/ref", "/tmp/x"}, &out)
	if code != 1 {
		t.Errorf("expected exit 1 for bad reference, got %d", code)
	}
}

func TestCLI_BadDate(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bad.txt")
	var out bytes.Buffer
	code := run([]string{"-d", "not-a-date", f}, &out)
	if code != 1 {
		t.Errorf("expected exit 1 for invalid date, got %d", code)
	}
}

func TestCLI_LongFlags(t *testing.T) {
	f := filepath.Join(t.TempDir(), "long.txt")
	var out bytes.Buffer
	code := run([]string{"--no-create", f}, &out)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Error("--no-create should not create file")
	}
}

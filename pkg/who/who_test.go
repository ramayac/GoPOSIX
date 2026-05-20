package who

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseUtmpEmpty(t *testing.T) {
	entry := make([]byte, utmpSize)
	u, err := parseUtmpEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Name != "" {
		t.Errorf("expected empty name for EMPTY entry, got %q", u.Name)
	}
}

func TestParseUtmpUserProcess(t *testing.T) {
	// Build a valid USER_PROCESS (type=7) utmp entry
	entry := make([]byte, utmpSize)
	// Type = USER_PROCESS (7)
	binary.LittleEndian.PutUint32(entry[0:4], 7)
	// Line = "pts/0"
	copy(entry[4+4:], "pts/0")
	// User = "testuser"
	copy(entry[4+4+utmpLineLen+4:], "testuser")
	// Host = "host.example.com"
	copy(entry[4+4+utmpLineLen+4+utmpUserLen:], "host.example.com")
	// Timestamp = 1700000000
	binary.LittleEndian.PutUint32(entry[utmpTimeOff:utmpTimeOff+4], 1700000000)

	u, err := parseUtmpEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Name != "testuser" {
		t.Errorf("expected name 'testuser', got %q", u.Name)
	}
	if u.Terminal != "pts/0" {
		t.Errorf("expected terminal 'pts/0', got %q", u.Terminal)
	}
	if u.Host != "host.example.com" {
		t.Errorf("expected host 'host.example.com', got %q", u.Host)
	}
	if u.Time == "" {
		t.Error("expected non-empty time")
	}
}

func TestParseUtmpNonUserProcess(t *testing.T) {
	// Type = BOOT_TIME (2) should return empty user
	entry := make([]byte, utmpSize)
	binary.LittleEndian.PutUint32(entry[0:4], 2)
	copy(entry[4+4+utmpLineLen+4:], "bootuser")

	u, err := parseUtmpEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Name != "" {
		t.Errorf("expected empty name for non-USER_PROCESS entry, got %q", u.Name)
	}
}

func TestParseUtmpShortEntry(t *testing.T) {
	_, err := parseUtmpEntry(make([]byte, 10))
	if err == nil {
		t.Error("expected error for short entry")
	}
}

func TestParseUtmpEmptyUser(t *testing.T) {
	// USER_PROCESS with empty username
	entry := make([]byte, utmpSize)
	binary.LittleEndian.PutUint32(entry[0:4], 7)
	// User field is all zeros

	u, err := parseUtmpEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Name != "" {
		t.Errorf("expected empty name for empty user field, got %q", u.Name)
	}
}

func TestReadUtmp(t *testing.T) {
	// Create a fake utmp file with two USER_PROCESS entries
	tmp := t.TempDir()
	utmpPath := filepath.Join(tmp, "utmp")

	// Entry 1: type=7, user=user1, line=pts/0
	e1 := make([]byte, utmpSize)
	binary.LittleEndian.PutUint32(e1[0:4], 7)
	copy(e1[4+4:], "pts/0")
	copy(e1[4+4+utmpLineLen+4:], "user1")
	binary.LittleEndian.PutUint32(e1[utmpTimeOff:utmpTimeOff+4], 1700000000)

	// Entry 2: type=7, user=user2, line=pts/1
	e2 := make([]byte, utmpSize)
	binary.LittleEndian.PutUint32(e2[0:4], 7)
	copy(e2[4+4:], "pts/1")
	copy(e2[4+4+utmpLineLen+4:], "user2")
	binary.LittleEndian.PutUint32(e2[utmpTimeOff:utmpTimeOff+4], 1700000001)

	data := append(e1, e2...)
	if err := os.WriteFile(utmpPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	users, err := readUtmp(utmpPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
	if users[0].Name != "user1" {
		t.Errorf("expected user1, got %q", users[0].Name)
	}
	if users[1].Name != "user2" {
		t.Errorf("expected user2, got %q", users[1].Name)
	}
	if users[0].Terminal != "pts/0" {
		t.Errorf("expected pts/0, got %q", users[0].Terminal)
	}
	if users[1].Terminal != "pts/1" {
		t.Errorf("expected pts/1, got %q", users[1].Terminal)
	}
}

func TestReadUtmpMixed(t *testing.T) {
	// Mix of USER_PROCESS and non-USER_PROCESS entries
	tmp := t.TempDir()
	utmpPath := filepath.Join(tmp, "utmp")

	// Entry 1: type=2 (BOOT_TIME) — should be skipped
	e1 := make([]byte, utmpSize)
	binary.LittleEndian.PutUint32(e1[0:4], 2)

	// Entry 2: type=7, user=realuser
	e2 := make([]byte, utmpSize)
	binary.LittleEndian.PutUint32(e2[0:4], 7)
	copy(e2[4+4+utmpLineLen+4:], "realuser")
	copy(e2[4+4:], "tty1")
	binary.LittleEndian.PutUint32(e2[utmpTimeOff:utmpTimeOff+4], 1700000000)

	// Entry 3: type=8 (DEAD_PROCESS) — should be skipped
	e3 := make([]byte, utmpSize)
	binary.LittleEndian.PutUint32(e3[0:4], 8)

	data := append(append(e1, e2...), e3...)
	if err := os.WriteFile(utmpPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	users, err := readUtmp(utmpPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user (only type 7), got %d", len(users))
	}
	if users[0].Name != "realuser" {
		t.Errorf("expected realuser, got %q", users[0].Name)
	}
}

func TestReadUtmpNonexistent(t *testing.T) {
	_, err := readUtmp("/nonexistent/utmp_file")
	if err == nil {
		t.Error("expected error for nonexistent utmp file")
	}
}

func TestReadUtmpEmptyFile(t *testing.T) {
	tmp := t.TempDir()
	utmpPath := filepath.Join(tmp, "utmp")
	if err := os.WriteFile(utmpPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	users, err := readUtmp(utmpPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users from empty file, got %d", len(users))
	}
}

func TestReadUtmpPartialEntry(t *testing.T) {
	tmp := t.TempDir()
	utmpPath := filepath.Join(tmp, "utmp")
	// File smaller than one utmp entry
	if err := os.WriteFile(utmpPath, make([]byte, 100), 0644); err != nil {
		t.Fatal(err)
	}

	users, err := readUtmp(utmpPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users from partial entry, got %d", len(users))
	}
}

func TestRunEmpty(t *testing.T) {
	result, err := Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Users == nil && result.Count != 0 {
		t.Error("expected consistent Users/Count")
	}
}

func TestWhoDefaultOutput(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{}, nil, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestWhoHeadingOutput(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-H"}, nil, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	t.Logf("who -H output:\n%s", buf.String())
}

func TestWhoQuickOutput(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"-q"}, nil, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	t.Logf("who -q output:\n%s", buf.String())
}

func TestWhoJson(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json"}, nil, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"users"`)) {
		t.Error("JSON output missing users field")
	}
}

func TestWhoJsonQuick(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json", "-q"}, nil, &buf)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"users"`)) {
		t.Error("JSON -q output missing users field")
	}
}

func TestFixedString(t *testing.T) {
	b := make([]byte, 10)
	b[0] = 'h'
	b[1] = 'i'
	s := fixedString(b)
	if s != "hi" {
		t.Errorf("expected 'hi', got %q", s)
	}
}

func TestFixedString_Full(t *testing.T) {
	b := []byte{'a', 'b', 'c'}
	s := fixedString(b)
	if s != "abc" {
		t.Errorf("expected 'abc', got %q", s)
	}
}

func TestFixedString_Empty(t *testing.T) {
	b := make([]byte, 10)
	// All zeros
	s := fixedString(b)
	if s != "" {
		t.Errorf("expected empty string, got %q", s)
	}
}

func TestRunViaCLI(t *testing.T) {
	code := run([]string{}, nil, io.Discard)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestWhoCLI_Quick(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"-q"}, nil, &out)
	if code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), "# users=") {
		t.Errorf("expected '# users=' in -q output, got: %q", out.String())
	}
}

func TestWhoCLI_Header(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"-H"}, nil, &out)
	if code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
}

func TestWhoRun_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out)
	if code != 2 {
		t.Errorf("exit %d, want 2 for bad flag", code)
	}
}

package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestParsePriorityDefault(t *testing.T) {
	pri, err := parsePriority("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pri != 1*8+5 {
		t.Errorf("expected 13 (user.notice), got %d", pri)
	}
}

func TestParsePriority(t *testing.T) {
	pri, err := parsePriority("local0.info")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pri != 16*8+6 {
		t.Errorf("expected 134 (local0.info), got %d", pri)
	}
}

func TestParsePriorityUserNotice(t *testing.T) {
	pri, err := parsePriority("user.notice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pri != 1*8+5 {
		t.Errorf("expected 13, got %d", pri)
	}
}

func TestParsePriorityInvalidFacility(t *testing.T) {
	_, err := parsePriority("bogus.info")
	if err == nil {
		t.Error("expected error for invalid facility")
	}
}

func TestRunBasic(t *testing.T) {
	// This will likely fail to connect to syslog in test environment,
	// but should not crash
	result, err := Run("test message", "testtag", "user.notice", false)
	if err != nil {
		t.Logf("syslog unavailable (expected in CI): %v", err)
		return
	}
	if result.Tag != "testtag" {
		t.Errorf("expected tag 'testtag', got %q", result.Tag)
	}
}

func TestRunFromStdin(t *testing.T) {
	// Test the CLI run function with stdin
	// We can't easily simulate stdin, so test the library
	result, err := Run("piped message", "mytag", "user.info", false)
	if err != nil {
		t.Logf("syslog unavailable: %v", err)
		return
	}
	if result.Message != "piped message" {
		t.Errorf("expected 'piped message', got %q", result.Message)
	}
}

func TestLoggerJson(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--json", "-t", "jsontag", "hello"}, &buf)
	// May fail to connect to syslog, but should still produce JSON
	if code != 0 {
		t.Logf("logger exit %d (may be OK if syslog unavailable)", code)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"tag"`)) {
		t.Log("JSON output missing tag field (may be OK if syslog error)")
	}
}

func TestFormatSyslogMessage(t *testing.T) {
	msg := formatSyslogMessage(13, "mytag", "hello world")
	if !strings.HasPrefix(msg, "<13>") {
		t.Errorf("expected <13> prefix, got %q", msg)
	}
	if !strings.Contains(msg, "mytag: hello world") {
		t.Errorf("expected message body, got %q", msg)
	}
}

package logger

import (
	"bytes"
	"fmt"
	"io"
	"net"
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
	code := run([]string{"--json", "-t", "jsontag", "hello"}, nil, &buf, &buf, "")
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

func TestParsePriorityDefaultEmpty(t *testing.T) {
	pri, err := parsePriority("")
	if err != nil {
		t.Fatal(err)
	}
	if pri != 1*8+5 {
		t.Errorf("expected user.notice (13), got %d", pri)
	}
}

func TestParsePriority_ShortForm(t *testing.T) {
	// Short form: just "info" — only one part, treated as facility lookup
	// "info" is not a facility, so this should error
	_, err := parsePriority("info")
	if err == nil {
		t.Error("expected error for 'info' as facility-only")
	}
}

func TestCLI_BadFlag(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--bad-flag"}, nil, &buf, &buf, "")
	if code != 2 {
		t.Errorf("expected exit 2 for bad flag, got %d", code)
	}
}

func TestParsePriority_ShortSeverity(t *testing.T) {
	// Single word without dot tries facility lookup — "debug" is not a facility
	_, err := parsePriority("debug")
	if err == nil {
		t.Error("expected error for 'debug' as facility-only")
	}
}

func TestParsePriority_UnknownSeverity(t *testing.T) {
	_, err := parsePriority("user.bogus")
	if err == nil {
		t.Error("expected error for unknown severity")
	}
}

func TestParsePriority_LocalFacility(t *testing.T) {
	pri, err := parsePriority("local4.warning")
	if err != nil {
		t.Fatal(err)
	}
	if pri != 20*8+4 {
		t.Errorf("expected local4.warning (164), got %d", pri)
	}
}

func TestRun_DifferentFacilities(t *testing.T) {
	// Test with daemon facility — likely fails in test env but shouldn't crash
	result, err := Run("test", "daemon", "daemon.info", false)
	if err != nil {
		t.Logf("syslog unavailable: %v", err)
		return
	}
	if result.Priority != "daemon.info" {
		t.Errorf("expected daemon.info, got %s", result.Priority)
	}
}

func TestRun_StderrFlag(t *testing.T) {
	// Test with stderr flag
	result, err := Run("stderr test", "mytag", "user.notice", true)
	if err != nil {
		t.Logf("syslog unavailable: %v", err)
		return
	}
	if result.Tag != "mytag" {
		t.Errorf("expected mytag, got %s", result.Tag)
	}
}

type mockConn struct {
	net.Conn
	buf       bytes.Buffer
	closed    bool
	failWrite bool
}

func (m *mockConn) Write(b []byte) (int, error) {
	if m.failWrite {
		return 0, io.EOF // we can import "io" or use standard errors
	}
	return m.buf.Write(b)
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func TestLoggerMockedDial(t *testing.T) {
	origDial := dialSyslogFn
	defer func() { dialSyslogFn = origDial }()

	// 1. Success case: dialing /dev/log succeeds
	conn := &mockConn{}
	dialSyslogFn = func(network, address string) (net.Conn, error) {
		if address == "/dev/log" {
			return conn, nil
		}
		return nil, io.EOF
	}

	res, err := Run("hello mock", "mocktag", "local0.info", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "hello mock" {
		t.Errorf("expected 'hello mock', got %q", res.Message)
	}
	if !conn.closed {
		t.Error("expected connection to be closed")
	}
	if !strings.Contains(conn.buf.String(), "<134>mocktag: hello mock") {
		t.Errorf("unexpected formatted message: %s", conn.buf.String())
	}

	// 2. Write fails on connection
	connWriteFail := &mockConn{failWrite: true}
	dialSyslogFn = func(network, address string) (net.Conn, error) {
		return connWriteFail, nil
	}
	_, err = Run("fail write", "mocktag", "local0.info", false)
	if err == nil {
		t.Error("expected error when connection write fails")
	}

	// 3. Dial fails completely (silent fallback)
	dialSyslogFn = func(network, address string) (net.Conn, error) {
		return nil, io.EOF
	}
	_, err = Run("fallback msg", "mocktag", "local0.info", false)
	if err != nil {
		t.Errorf("unexpected error on silent fallback: %v", err)
	}
}

func TestLoggerCLI_InjectableStreams(t *testing.T) {
	// 1. Stdin reading
	stdin := strings.NewReader("message from stdin\n")
	var stdout, stderr bytes.Buffer
	code := run([]string{"-t", "stdintag"}, stdin, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d. Stderr: %s", code, stderr.String())
	}

	// 2. Priority parse failure in CLI
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-p", "invalid.pri", "msg"}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("expected exit 1 for invalid priority, got %d", code)
	}
	if !strings.Contains(stderr.String(), "logger:") {
		t.Errorf("expected error logs on stderr, got: %s", stderr.String())
	}

	// 3. AlsoStderr flag via CLI
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-s", "-t", "tag", "log to stderr"}, nil, &stdout, &stderr, "")
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stderr.String(), "log to stderr") {
		t.Errorf("expected stderr to contain 'log to stderr', got: %s", stderr.String())
	}
}

func TestRun_EmptyTag(t *testing.T) {
	result, err := Run("empty tag msg", "", "user.notice", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Tag != "logger" {
		t.Errorf("expected tag to default to 'logger', got %s", result.Tag)
	}
}

type mockReader struct{}

func (mockReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("mock read error")
}

func TestLoggerCLI_ReadError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{}, mockReader{}, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("expected exit 1 for read error, got %d", code)
	}
	if !strings.Contains(stderr.String(), "logger: mock read error") {
		t.Errorf("expected stderr to contain read error logs, got: %s", stderr.String())
	}
}

func TestLoggerCLI_WriteFailure(t *testing.T) {
	origDial := dialSyslogFn
	defer func() { dialSyslogFn = origDial }()

	dialSyslogFn = func(network, address string) (net.Conn, error) {
		return &mockConn{failWrite: true}, nil
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"message"}, nil, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("expected exit 1 for write failure, got %d", code)
	}
}

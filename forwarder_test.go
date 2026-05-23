package goposix

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ramayac/goposix/internal/daemon"
	_ "github.com/ramayac/goposix/pkg/echo"
)

func TestSocketExists(t *testing.T) {
	// Non-existent socket.
	if socketExists("/tmp/nonexistent-goposix-test-sock") {
		t.Error("socketExists should return false for nonexistent path")
	}

	// Regular file should not count as socket.
	f, err := os.CreateTemp("", "goposix-test-file")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Close()
	if socketExists(f.Name()) {
		t.Error("socketExists should return false for regular file")
	}

	// Non-existent path.
	if socketExists("") {
		t.Error("socketExists should return false for empty path")
	}
}

func TestIsStdinPiped(t *testing.T) {
	// This is hard to test in unit tests since stdin varies.
	// Just verify the function doesn't panic.
	result := isStdinPiped()
	t.Logf("isStdinPiped = %v (depends on test runner)", result)
}

func TestTryForwardNoDaemon(t *testing.T) {
	// No daemon running — should return -1 (fallback to cold start).
	os.Setenv("GOPOSIX_SOCKET", "/tmp/nonexistent-goposix-m5-test-sock")
	defer os.Unsetenv("GOPOSIX_SOCKET")

	code := TryForward()
	if code != -1 {
		t.Errorf("TryForward with no daemon should return -1, got %d", code)
	}
}

func startTestDaemon(t *testing.T) string {
	socketPath := filepath.Join(t.TempDir(), "goposix.sock")

	// Start daemon in background
	go func() {
		err := daemon.RunDaemon(socketPath, 2, "")
		if err != nil {
			t.Logf("daemon exited: %v", err)
		}
	}()

	// Wait for socket to be created
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			return socketPath
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("daemon socket not created in time")
	return ""
}

func TestTryForward_PipedStdin(t *testing.T) {
	// When stdin is piped, TryForward should return -1 (fallback to cold start).
	// We mock isStdinPipedFn to simulate piped stdin.
	origFn := isStdinPipedFn
	isStdinPipedFn = func() bool { return true }
	defer func() { isStdinPipedFn = origFn }()

	// Point GOPOSIX_SOCKET to a valid-looking path (but it shouldn't matter
	// since piped stdin short-circuits).
	os.Setenv("GOPOSIX_SOCKET", "/tmp/should-not-check-socket")
	defer os.Unsetenv("GOPOSIX_SOCKET")

	code := TryForward()
	if code != -1 {
		t.Errorf("TryForward with piped stdin should return -1, got %d", code)
	}
}

func TestForwardToDaemon_NoSocket(t *testing.T) {
	// forwardToDaemon with a non-existent socket should return -1.
	code := forwardToDaemon("/tmp/nonexistent-goposix-forward-test.sock",
		[]string{"goposix", "echo", "hello"})
	if code != -1 {
		t.Errorf("expected -1 (fallback), got %d", code)
	}
}

func TestForwardToDaemon_MetaFlags(t *testing.T) {
	// When a well-known binary is invoked with --help, --version, or
	// --list-commands, forwardToDaemon should return -1 to fall back.
	for _, flag := range []string{"--help", "-h", "--version", "--list-commands"} {
		code := forwardToDaemon("/tmp/irrelevant.sock",
			[]string{"goposix", flag})
		if code != -1 {
			t.Errorf("forwardToDaemon with %s should return -1, got %d", flag, code)
		}
	}
}

func TestForwardToDaemon_MarshalError(t *testing.T) {
	// forwardToDaemon with a non-marshallable argument should return 126.
	// Actually, Go JSON marshaling doesn't fail on strings/ints/bools.
	// The marshal path is hard to trigger, but verify the function exists.
	// For coverage: send a command name that can't be marshaled (like NaN).
	// Since we use simple structs, we can't easily trigger a marshal error.
	// Instead, we test the connection failure path which covers the fallback.
	code := forwardToDaemon("/tmp/definitely-not-a-socket",
		[]string{"goposix", "echo", "test"})
	if code != -1 {
		t.Errorf("expected -1 for connection failure, got %d", code)
	}
}

func TestTryForward_Integration(t *testing.T) {
	// Spin up a test daemon
	socketPath := startTestDaemon(t)

	// Set socket environment variable
	os.Setenv("GOPOSIX_SOCKET", socketPath)
	defer os.Unsetenv("GOPOSIX_SOCKET")

	// Set hooks
	origStdinFn := isStdinPipedFn
	isStdinPipedFn = func() bool { return false }
	defer func() { isStdinPipedFn = origStdinFn }()

	var buf bytes.Buffer
	origStdout := stdoutWriter
	stdoutWriter = &buf
	defer func() { stdoutWriter = origStdout }()

	// Mock arguments to invoke `echo hello m5`
	origArgs := os.Args
	os.Args = []string{"goposix", "echo", "hello", "m5"}
	defer func() { os.Args = origArgs }()

	// Execute TryForward
	exitCode := TryForward()
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Verify the captured stdout matches the expected echo output
	expected := "hello m5\n"
	if buf.String() != expected {
		t.Errorf("expected stdout %q, got %q", expected, buf.String())
	}
}

func TestForwardToDaemon_WellKnownShortArgv(t *testing.T) {
	code := forwardToDaemon("/tmp/irrelevant.sock", []string{"goposix"})
	if code != -1 {
		t.Errorf("expected -1 for short argv on well-known, got %d", code)
	}
}

func TestTryForward_DefaultSocketPath(t *testing.T) {
	// Unset socket env
	origEnv := os.Getenv("GOPOSIX_SOCKET")
	os.Unsetenv("GOPOSIX_SOCKET")
	defer func() {
		if origEnv != "" {
			os.Setenv("GOPOSIX_SOCKET", origEnv)
		}
	}()

	// TryForward should fail and return -1 since /var/run/goposix.sock won't exist
	code := TryForward()
	if code != -1 {
		t.Errorf("expected -1, got %d", code)
	}
}

func TestForwardToDaemon_SocketErrorAndInvalidJson(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "goposix-mock.sock")
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// 1. Test invalid JSON response
	go func() {
		conn, err := l.Accept()
		if err == nil {
			// Read request
			buf := make([]byte, 1024)
			_, _ = conn.Read(buf)
			// Write invalid JSON
			_, _ = conn.Write([]byte("invalid json\n"))
			conn.Close()
		}
	}()

	code := forwardToDaemon(socketPath, []string{"goposix", "echo", "test"})
	if code != 126 {
		t.Errorf("expected 126 for invalid JSON, got %d", code)
	}

	// 2. Test JSON-RPC error response
	go func() {
		conn, err := l.Accept()
		if err == nil {
			buf := make([]byte, 1024)
			_, _ = conn.Read(buf)
			// Write error JSON-RPC
			_, _ = conn.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found"}}` + "\n"))
			conn.Close()
		}
	}()

	code = forwardToDaemon(socketPath, []string{"goposix", "echo", "test"})
	if code != 126 {
		t.Errorf("expected 126 for JSON-RPC error, got %d", code)
	}
}


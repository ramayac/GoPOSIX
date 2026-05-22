package goposix

import (
	"bytes"
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

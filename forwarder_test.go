package goposix

import (
	"os"
	"testing"
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

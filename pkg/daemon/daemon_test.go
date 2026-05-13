package daemon

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ramayac/korego/internal/daemon"
)

func TestRunDaemonSocketCreation(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "korego.sock")
	srv := daemon.NewServer(socket, 2, "")
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()
	time.Sleep(50 * time.Millisecond)
}

func TestRunDaemonGracefulShutdown(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "korego.sock")
	srv := daemon.NewServer(socket, 1, "")
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	srv.Stop()
}

func TestRunDaemonWorkerCount(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "korego.sock")
	// Workers > 1 should work fine
	srv := daemon.NewServer(socket, 8, "")
	if err := srv.Start(); err != nil {
		t.Fatalf("Start with 8 workers: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	srv.Stop()
}

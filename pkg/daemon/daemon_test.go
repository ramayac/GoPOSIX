package daemon

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/ramayac/goposix/internal/daemon"
)

func TestRunDaemonSocketCreation(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "goposix.sock")
	srv := daemon.NewServer(socket, 2, "")
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()
	time.Sleep(50 * time.Millisecond)
}

func TestRunDaemonGracefulShutdown(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "goposix.sock")
	srv := daemon.NewServer(socket, 1, "")
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	srv.Stop()
}

func TestRunDaemonWorkerCount(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "goposix.sock")
	// Workers > 1 should work fine
	srv := daemon.NewServer(socket, 8, "")
	if err := srv.Start(); err != nil {
		t.Fatalf("Start with 8 workers: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	srv.Stop()
}

// --- CLI layer tests ---

func TestCLI_BadFlag(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--nonexistent"}, nil, &out)
	if code != 2 {
		t.Errorf("expected exit 2 for bad flag, got %d", code)
	}
}

func TestCLI_WorkerCount(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "goposix.sock")
	var out bytes.Buffer
	go func() {
		run([]string{"-s", socket, "-w", "2"}, nil, &out)
	}()
	time.Sleep(100 * time.Millisecond)
}

func TestCLI_DefaultSocketFlag(t *testing.T) {
	var out bytes.Buffer
	// -s with a temp socket path — daemon will bind to it
	socket := filepath.Join(t.TempDir(), "def.sock")
	go func() {
		run([]string{"-s", socket}, nil, &out)
	}()
	time.Sleep(100 * time.Millisecond)
}

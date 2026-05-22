package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ramayac/goposix/internal/daemon"
	"github.com/ramayac/goposix/pkg/client"

	// Register all utilities so the daemon can dispatch them
	_ "github.com/ramayac/goposix/pkg/cat"
	_ "github.com/ramayac/goposix/pkg/echo"
	_ "github.com/ramayac/goposix/pkg/ls"
	_ "github.com/ramayac/goposix/pkg/pwd"
)

func TestDaemonConcurrent(t *testing.T) {
	t.Setenv("GOPOSIX_RATE_LIMIT", "10000")
	socket := filepath.Join(t.TempDir(), "goposix.sock")

	// Start server in background
	server := daemon.NewServer(socket, 4, "")
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Wait for socket to be created
	time.Sleep(100 * time.Millisecond)

	c := client.Dial(socket, 2*time.Second)

	// Single ping test
	var pingRes map[string]interface{}
	ctx := context.Background()
	err := c.Call(ctx, "goposix.ping", nil, &pingRes)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if pingRes["pong"] != true {
		t.Errorf("Expected pong=true, got %v", pingRes["pong"])
	}

	// Single ls test
	var lsRes map[string]interface{}
	err = c.Call(ctx, "goposix.ls", map[string]interface{}{"path": "/tmp"}, &lsRes)
	if err != nil {
		t.Fatalf("ls failed: %v", err)
	}
	if lsRes["files"] == nil {
		// Just ensure it ran
	}

	// Concurrent test
	var wg sync.WaitGroup
	numRequests := 100
	errs := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var res map[string]interface{}
			err := c.Call(ctx, "goposix.echo", map[string]interface{}{"text": fmt.Sprintf("req-%d", idx)}, &res)
			if err != nil {
				errs <- err
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	if len(errs) > 0 {
		t.Fatalf("%d requests failed. First error: %v", len(errs), <-errs)
	}
}

func TestDaemonBatch(t *testing.T) {
	// Our client currently only supports single Call().
	// To test batch, we could use net.Dial directly, or modify the client.
	// We'll use net.Dial for a quick check.
	socket := filepath.Join(t.TempDir(), "goposix-batch.sock")
	server := daemon.NewServer(socket, 4, "")
	server.Start()
	defer server.Stop()
	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	batchReq := `[
		{"jsonrpc":"2.0", "method":"goposix.echo", "params":{"text":"a"}, "id":1},
		{"jsonrpc":"2.0", "method":"goposix.echo", "params":{"text":"b"}, "id":2}
	]`
	conn.Write([]byte(batchReq))

	var res []map[string]interface{}
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&res); err != nil {
		t.Fatal(err)
	}

	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
}

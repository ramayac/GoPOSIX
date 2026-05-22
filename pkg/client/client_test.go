package client

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ramayac/goposix/internal/daemon"

	// Register utilities needed by tests.
	_ "github.com/ramayac/goposix/pkg/basename"
	_ "github.com/ramayac/goposix/pkg/cat"
	_ "github.com/ramayac/goposix/pkg/diff"
	_ "github.com/ramayac/goposix/pkg/echo"
	_ "github.com/ramayac/goposix/pkg/grep"
	_ "github.com/ramayac/goposix/pkg/head"
	_ "github.com/ramayac/goposix/pkg/ls"
	_ "github.com/ramayac/goposix/pkg/mkdir"
	_ "github.com/ramayac/goposix/pkg/pwd"
	_ "github.com/ramayac/goposix/pkg/rm"
	_ "github.com/ramayac/goposix/pkg/stat"
	_ "github.com/ramayac/goposix/pkg/tail"
	_ "github.com/ramayac/goposix/pkg/touch"
	_ "github.com/ramayac/goposix/pkg/truefalse"
	_ "github.com/ramayac/goposix/pkg/wc"
)

func startDaemon(t *testing.T) (string, func()) {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "goposix-test.sock")
	srv := daemon.NewServer(socket, 4, "")
	if err := srv.Start(); err != nil {
		t.Fatalf("start daemon: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	return socket, func() { srv.Stop() }
}

func TestPing(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, err := New(socket)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	res, err := c.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if !res.Pong {
		t.Error("expected pong=true")
	}
	if res.Version == "" {
		t.Error("expected non-empty version")
	}
}

func TestEcho(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, err := New(socket)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	res, err := c.Echo(ctx, "hello test")
	if err != nil {
		t.Fatalf("Echo: %v", err)
	}
	if res.Text != "hello test" {
		t.Errorf("expected 'hello test', got %q", res.Text)
	}
}

func TestLs(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, err := New(socket)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	res, err := c.Ls(ctx, "/tmp", nil)
	if err != nil {
		t.Fatalf("Ls: %v", err)
	}
	if res.Path != "/tmp" {
		t.Errorf("expected path /tmp, got %q", res.Path)
	}
	if res.Total < 0 {
		t.Errorf("expected non-negative total, got %d", res.Total)
	}
}

func TestPwd(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket)
	defer c.Close()

	res, err := c.Pwd(context.Background())
	if err != nil {
		t.Fatalf("Pwd: %v", err)
	}
	if res.Path == "" {
		t.Error("expected non-empty path")
	}
}

func TestWc(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket)
	defer c.Close()

	res, err := c.Wc(context.Background(), "/etc/hosts")
	if err != nil {
		t.Fatalf("Wc: %v", err)
	}
	if res.Lines < 1 {
		t.Errorf("expected at least 1 line, got %d", res.Lines)
	}
}

func TestSessionLifecycle(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket)
	defer c.Close()
	ctx := context.Background()

	s, err := c.SessionCreate(ctx)
	if err != nil {
		t.Fatalf("SessionCreate: %v", err)
	}
	if s.SessionID == "" {
		t.Error("expected non-empty session ID")
	}

	if err := c.SessionSetCwd(ctx, s.SessionID, "/tmp"); err != nil {
		t.Fatalf("SessionSetCwd: %v", err)
	}

	list, err := c.SessionList(ctx)
	if err != nil {
		t.Fatalf("SessionList: %v", err)
	}
	found := false
	for _, si := range list {
		if si.SessionID == s.SessionID {
			found = true
			break
		}
	}
	if !found {
		t.Error("session not found in list")
	}

	if err := c.SessionDestroy(ctx, s.SessionID); err != nil {
		t.Fatalf("SessionDestroy: %v", err)
	}
}

func TestShellExec(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket)
	defer c.Close()
	ctx := context.Background()

	s, _ := c.SessionCreate(ctx)
	res, err := c.ShellExec(ctx, s.SessionID, "echo hello world")
	if err != nil {
		t.Fatalf("ShellExec: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", res.ExitCode)
	}
}

func TestBatch(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket)
	defer c.Close()
	ctx := context.Background()

	reqs := []BatchRequest{
		{Method: "goposix.echo", Params: map[string]string{"text": "a"}},
		{Method: "goposix.echo", Params: map[string]string{"text": "b"}},
		{Method: "goposix.ping", Params: nil},
	}

	resps, err := c.Batch(ctx, reqs)
	if err != nil {
		t.Fatalf("Batch: %v", err)
	}
	if len(resps) != 3 {
		t.Fatalf("expected 3 responses, got %d", len(resps))
	}
	if resps[0].Error != nil {
		t.Errorf("req 0 error: %v", resps[0].Error)
	}
	if resps[1].Error != nil {
		t.Errorf("req 1 error: %v", resps[1].Error)
	}
	if resps[2].Error != nil {
		t.Errorf("req 2 error: %v", resps[2].Error)
	}
}

func TestNotification(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket)
	defer c.Close()
	ctx := context.Background()

	// Notifications receive no response — just verify no error.
	if err := c.Notify(ctx, "goposix.true", nil); err != nil {
		t.Fatalf("Notify: %v", err)
	}
}

func TestCallRaw(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket)
	defer c.Close()
	ctx := context.Background()

	raw, err := c.CallRaw(ctx, "goposix.ping", nil)
	if err != nil {
		t.Fatalf("CallRaw: %v", err)
	}
	if len(raw) == 0 {
		t.Error("expected non-empty raw result")
	}
}

func TestErrorMethodNotFound(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket)
	defer c.Close()
	ctx := context.Background()

	err := c.Call(ctx, "goposix.nonexistent", nil, nil)
	if err == nil {
		t.Fatal("expected error for unknown method")
	}
	var rpcErr *rpcError
	if !errors.As(err, &rpcErr) {
		t.Errorf("expected rpcError, got %T: %v", err, err)
	}
}

func TestConnectionPoolReuse(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket, WithPoolSize(2))
	defer c.Close()
	ctx := context.Background()

	// Make multiple calls to exercise pool reuse.
	for i := 0; i < 10; i++ {
		_, err := c.Ping(ctx)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
}

func TestConnectionPoolExhaustion(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket, WithPoolSize(2), WithTimeout(10*time.Second))
	defer c.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	errs := make(chan error, 8)

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
			_, err := c.Ping(ctx)
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("unexpected error under pool pressure: %v", err)
	}
}

func TestContextCancellation(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket, WithTimeout(5*time.Second))
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := c.Ping(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestContextTimeout(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket, WithPoolSize(1), WithTimeout(5*time.Second))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Exhaust the pool so the next call blocks.
	c.pool.sem <- struct{}{} // hold the slot

	_, err := c.Ping(ctx)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	<-c.pool.sem // release
}

func TestStat(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket)
	defer c.Close()

	res, err := c.Stat(context.Background(), "/etc/hosts")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if res.Path != "/etc/hosts" {
		t.Errorf("expected /etc/hosts, got %q", res.Path)
	}
	if res.Size == 0 {
		t.Error("expected non-zero size")
	}
}

func TestCallGeneric(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket)
	defer c.Close()

	// Test the generic Call method for backward compatibility.
	var result map[string]interface{}
	err := c.Call(context.Background(), "goposix.ping", nil, &result)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result["pong"] != true {
		t.Error("expected pong=true")
	}
}

func TestDiff(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket)
	defer c.Close()

	res, err := c.Diff(context.Background(), "/etc/hosts", "/etc/host.conf")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(res.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(res.Files))
	}
}

func TestGrep(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket)
	defer c.Close()

	matches, err := c.Grep(context.Background(), "localhost", []string{"/etc/hosts"})
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if len(matches) == 0 {
		t.Error("expected at least one match for 'localhost' in /etc/hosts")
	}
}

func TestBasename(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()

	c, _ := New(socket)
	defer c.Close()

	res, err := c.Basename(context.Background(), "/etc/hosts")
	if err != nil {
		t.Fatalf("Basename: %v", err)
	}
	if res.Result != "hosts" {
		t.Errorf("expected 'hosts', got %q", res.Result)
	}
}

func TestCat(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()
	c, _ := New(socket)
	defer c.Close()
	res, err := c.Cat(context.Background(), "/etc/hosts")
	if err != nil {
		t.Fatalf("Cat: %v", err)
	}
	if len(res.Lines) == 0 {
		t.Error("expected lines")
	}
}

func TestHead(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()
	c, _ := New(socket)
	defer c.Close()
	res, err := c.Head(context.Background(), "/etc/hosts", 3)
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	if len(res.Lines) == 0 {
		t.Error("expected lines")
	}
}

func TestTail(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()
	c, _ := New(socket)
	defer c.Close()
	res, err := c.Tail(context.Background(), "/etc/hosts", 3)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(res.Lines) == 0 {
		t.Error("expected lines")
	}
}

func TestRm(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "delme.txt")
	os.WriteFile(f, []byte("x"), 0644)
	socket, stop := startDaemon(t)
	defer stop()
	c, _ := New(socket)
	defer c.Close()
	res, err := c.Rm(context.Background(), []string{f}, true, false)
	if err != nil {
		t.Fatalf("Rm: %v", err)
	}
	if len(res.Removed) == 0 {
		t.Error("expected removed")
	}
}

func TestMkdir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "newdir")
	socket, stop := startDaemon(t)
	defer stop()
	c, _ := New(socket)
	defer c.Close()
	res, err := c.Mkdir(context.Background(), dir, false)
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if len(res.Created) == 0 {
		t.Error("expected created")
	}
}

func TestTouch(t *testing.T) {
	f := filepath.Join(t.TempDir(), "touchme.txt")
	socket, stop := startDaemon(t)
	defer stop()
	c, _ := New(socket)
	defer c.Close()
	res, err := c.Touch(context.Background(), []string{f})
	if err != nil {
		t.Fatalf("Touch: %v", err)
	}
	if len(res.Touched) == 0 {
		t.Error("expected touched")
	}
}

func TestWithMaxRetries(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()
	c, _ := New(socket, WithMaxRetries(2))
	defer c.Close()
	_, err := c.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping with retries: %v", err)
	}
}

func TestDial(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()
	_ = stop
	c2 := Dial(socket, 1*time.Second)
	if c2 == nil {
		t.Fatal("Dial returned nil")
	}
	c2.Close()
}

func TestCloseTwice(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()
	c, _ := New(socket)
	c.Close()
	c.Close() // should not panic
}

func TestContextCancelBeforeCall(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()
	c, _ := New(socket)
	defer c.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Ping(ctx)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestHelper_Stat(t *testing.T) {
	socket, stop := startDaemon(t)
	defer stop()
	c, _ := New(socket)
	defer c.Close()
	res, err := c.Stat(context.Background(), "/tmp")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !res.IsDir {
		t.Error("/tmp should be a directory")
	}
}

func TestRpcError_Error(t *testing.T) {
	e := &rpcError{Code: -32600, Message: "Invalid Request"}
	s := e.Error()
	if s != "RPC error -32600: Invalid Request" {
		t.Errorf("Error() = %q, want 'RPC error -32600: Invalid Request'", s)
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{-1, "-1"},
		{-10, "-10"},
		{100, "100"},
		{999, "999"},
		{-999, "-999"},
		{12345, "12345"},
	}
	for _, tt := range tests {
		got := itoa(tt.n)
		if got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

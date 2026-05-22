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
	_ "github.com/ramayac/goposix/pkg/chmod"
	_ "github.com/ramayac/goposix/pkg/chown"
	_ "github.com/ramayac/goposix/pkg/cp"
	_ "github.com/ramayac/goposix/pkg/cut"
	_ "github.com/ramayac/goposix/pkg/date"
	_ "github.com/ramayac/goposix/pkg/df"
	_ "github.com/ramayac/goposix/pkg/diff"
	_ "github.com/ramayac/goposix/pkg/dirname"
	_ "github.com/ramayac/goposix/pkg/du"
	_ "github.com/ramayac/goposix/pkg/echo"
	_ "github.com/ramayac/goposix/pkg/env"
	_ "github.com/ramayac/goposix/pkg/expr"
	_ "github.com/ramayac/goposix/pkg/find"
	_ "github.com/ramayac/goposix/pkg/grep"
	_ "github.com/ramayac/goposix/pkg/gzip"
	_ "github.com/ramayac/goposix/pkg/head"
	_ "github.com/ramayac/goposix/pkg/hostname"
	_ "github.com/ramayac/goposix/pkg/id"
	_ "github.com/ramayac/goposix/pkg/kill"
	_ "github.com/ramayac/goposix/pkg/ln"
	_ "github.com/ramayac/goposix/pkg/ls"
	_ "github.com/ramayac/goposix/pkg/md5sum"
	_ "github.com/ramayac/goposix/pkg/mkdir"
	_ "github.com/ramayac/goposix/pkg/mv"
	_ "github.com/ramayac/goposix/pkg/printenv"
	_ "github.com/ramayac/goposix/pkg/printf"
	_ "github.com/ramayac/goposix/pkg/ps"
	_ "github.com/ramayac/goposix/pkg/pwd"
	_ "github.com/ramayac/goposix/pkg/readlink"
	_ "github.com/ramayac/goposix/pkg/rm"
	_ "github.com/ramayac/goposix/pkg/rmdir"
	_ "github.com/ramayac/goposix/pkg/sha256sum"
	_ "github.com/ramayac/goposix/pkg/sort"
	_ "github.com/ramayac/goposix/pkg/stat"
	_ "github.com/ramayac/goposix/pkg/tail"
	_ "github.com/ramayac/goposix/pkg/tar"
	_ "github.com/ramayac/goposix/pkg/testcmd"
	_ "github.com/ramayac/goposix/pkg/touch"
	_ "github.com/ramayac/goposix/pkg/truefalse"
	_ "github.com/ramayac/goposix/pkg/uname"
	_ "github.com/ramayac/goposix/pkg/uniq"
	_ "github.com/ramayac/goposix/pkg/wc"
	_ "github.com/ramayac/goposix/pkg/whoami"
	_ "github.com/ramayac/goposix/pkg/xargs"
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

// --- Phase C: coverage push for all client helpers ---

func startDaemonForHelper(t *testing.T) (*Client, func()) {
	socket, stop := startDaemon(t)
	c, err := New(socket)
	if err != nil {
		stop()
		t.Fatalf("New: %v", err)
	}
	return c, func() { c.Close(); stop() }
}

func TestHelper_Dirname(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Dirname(context.Background(), "/usr/bin/ls")
	if err != nil {
		t.Fatalf("Dirname: %v", err)
	}
	if res.Result != "/usr/bin" {
		t.Errorf("got %q, want /usr/bin", res.Result)
	}
}

func TestHelper_Hostname(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Hostname(context.Background())
	if err != nil {
		t.Fatalf("Hostname: %v", err)
	}
	if res.Hostname == "" {
		t.Error("expected non-empty hostname")
	}
}

func TestHelper_Printf(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Printf(context.Background(), "hello %s", "world")
	if err != nil {
		t.Fatalf("Printf: %v", err)
	}
	if res.Output != "hello world" {
		t.Errorf("got %q, want 'hello world'", res.Output)
	}
}

func TestHelper_Test(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Test(context.Background(), []string{"-n", "hello"})
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if !res.Result {
		t.Error("expected true for -n hello")
	}
}

func TestHelper_Whoami(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Whoami(context.Background())
	if err != nil {
		t.Fatalf("Whoami: %v", err)
	}
	if res.User == "" {
		t.Error("expected non-empty username")
	}
}

func TestHelper_Readlink(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Readlink(context.Background(), "/proc/self")
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if res.Target == "" {
		t.Error("expected non-empty symlink target")
	}
}

func TestHelper_ID(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.ID(context.Background())
	if err != nil {
		t.Fatalf("ID: %v", err)
	}
	if res.UID < 0 {
		t.Error("expected non-negative UID")
	}
}

func TestHelper_Date(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Date(context.Background())
	if err != nil {
		t.Fatalf("Date: %v", err)
	}
	if res.ISO == "" {
		t.Error("expected non-empty date string")
	}
}

func TestHelper_Uname(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Uname(context.Background())
	if err != nil {
		t.Fatalf("Uname: %v", err)
	}
	if res.Sysname == "" {
		t.Error("expected non-empty sysname")
	}
}

func TestHelper_Env(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Env(context.Background(), nil, map[string]string{"GOPOSIX_TEST": "1"})
	if err != nil {
		t.Fatalf("Env: %v", err)
	}
	if len(res.Vars) == 0 {
		t.Error("expected non-empty vars")
	}
}

func TestHelper_Printenv(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Printenv(context.Background(), "PATH")
	if err != nil {
		t.Fatalf("Printenv: %v", err)
	}
	if res.Vars == nil {
		t.Error("expected non-nil vars")
	}
}

func TestHelper_Sort(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Sort(context.Background(), []string{"--json"})
	if err != nil {
		t.Fatalf("Sort: %v", err)
	}
	_ = res
}

func TestHelper_Cut(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Cut(context.Background(), []string{"-f1", "-d:"})
	if err != nil {
		t.Fatalf("Cut: %v", err)
	}
	_ = res
}

func TestHelper_Uniq(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	items, err := c.Uniq(context.Background(), []string{"--json"})
	if err != nil {
		t.Fatalf("Uniq: %v", err)
	}
	_ = items
}

func TestHelper_Find(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello"), 0644)
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	entries, err := c.Find(context.Background(), tmpDir, []string{"-name", "test.txt", "--json"})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one find result")
	}
}

func TestHelper_Mv(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	os.WriteFile(src, []byte("test"), 0644)
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Mv(context.Background(), src, dst)
	if err != nil {
		t.Fatalf("Mv: %v", err)
	}
	if len(res.Moved) == 0 {
		t.Error("expected at least one move record")
	}
}

func TestHelper_Cp(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	os.WriteFile(src, []byte("test"), 0644)
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Cp(context.Background(), src, dst)
	if err != nil {
		t.Fatalf("Cp: %v", err)
	}
	if len(res.Copied) == 0 {
		t.Error("expected at least one copy record")
	}
}

func TestHelper_Ln(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	os.WriteFile(src, []byte("test"), 0644)
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Ln(context.Background(), src, link, false)
	if err != nil {
		t.Fatalf("Ln: %v", err)
	}
	if len(res.Links) == 0 {
		t.Error("expected at least one link entry")
	}
}

func TestHelper_Rmdir(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "emptydir")
	os.MkdirAll(target, 0755)
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	_, err := c.Rmdir(context.Background(), target)
	if err != nil {
		t.Fatalf("Rmdir: %v", err)
	}
}

func TestHelper_Chmod(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	os.WriteFile(f, []byte("test"), 0644)
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	_, err := c.Chmod(context.Background(), "755", []string{f})
	if err != nil {
		t.Fatalf("Chmod: %v", err)
	}
}

func TestHelper_Chown(t *testing.T) {
	t.Skip("chown requires root privileges")
}

func TestHelper_Chgrp(t *testing.T) {
	t.Skip("chgrp requires root privileges")
}

func TestHelper_Md5sum(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	os.WriteFile(f, []byte("test"), 0644)
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	_, err := c.Md5sum(context.Background(), []string{f}, false)
	if err != nil {
		t.Fatalf("Md5sum: %v", err)
	}
}

func TestHelper_Sha256sum(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	os.WriteFile(f, []byte("test"), 0644)
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	_, err := c.Sha256sum(context.Background(), []string{f}, false)
	if err != nil {
		t.Fatalf("Sha256sum: %v", err)
	}
}

func TestHelper_Gzip(t *testing.T) {
	t.Skip("gzip helper needs specific stdin piping setup")
}

func TestHelper_Tar(t *testing.T) {
	t.Skip("tar helper needs absolute path resolution in daemon cwd")
}

func TestHelper_Df(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	info, err := c.Df(context.Background(), "/")
	if err != nil {
		t.Fatalf("Df: %v", err)
	}
	if len(info) == 0 {
		t.Error("expected at least one filesystem")
	}
}

func TestHelper_Du(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "f.txt"), []byte("test"), 0644)
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	entries, err := c.Du(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Du: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one entry")
	}
}

func TestHelper_Ps(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	procs, err := c.Ps(context.Background())
	if err != nil {
		t.Fatalf("Ps: %v", err)
	}
	if len(procs) == 0 {
		t.Error("expected at least one process")
	}
}

func TestHelper_Kill(t *testing.T) {
	t.Skip("kill requires actual pid and signal handling")
}

func TestHelper_Xargs(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	entries, err := c.Xargs(context.Background(), "echo", []string{"--json"})
	if err != nil {
		t.Fatalf("Xargs: %v", err)
	}
	_ = entries
}

func TestHelper_Expr(t *testing.T) {
	c, cleanup := startDaemonForHelper(t)
	defer cleanup()
	res, err := c.Expr(context.Background(), []string{"1", "+", "1"})
	if err != nil {
		t.Fatalf("Expr: %v", err)
	}
	if res.Result != "2" {
		t.Errorf("got %q, want '2'", res.Result)
	}
}

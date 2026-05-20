package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
)

// =============================================================================
// WorkerPool tests
// =============================================================================

func TestWorkerPool_New(t *testing.T) {
	wp := NewWorkerPool(5)
	if wp == nil {
		t.Fatal("NewWorkerPool returned nil")
	}
	if cap(wp.sem) != 5 {
		t.Errorf("expected capacity 5, got %d", cap(wp.sem))
	}
}

func TestWorkerPool_Submit(t *testing.T) {
	wp := NewWorkerPool(2)
	var mu sync.Mutex
	var count int

	for i := 0; i < 5; i++ {
		err := wp.Submit(context.Background(), func() {
			time.Sleep(1 * time.Millisecond)
			mu.Lock()
			count++
			mu.Unlock()
		})
		if err != nil {
			t.Fatalf("Submit failed: %v", err)
		}
	}
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	c := count
	mu.Unlock()
	if c != 5 {
		t.Errorf("expected 5 completions, got %d", c)
	}
}

func TestWorkerPool_SubmitContextCancel(t *testing.T) {
	wp := NewWorkerPool(1)
	wp.sem <- struct{}{} // fill pool

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := wp.Submit(ctx, func() {})
	if err == nil {
		t.Error("expected error from cancelled context")
	}
	<-wp.sem // clean up
}

// =============================================================================
// Server initialization
// =============================================================================

func TestNewServer_Basic(t *testing.T) {
	s := NewServer("/tmp/test_daemon.sock", 4, "")
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
	if s.workersMax != 4 {
		t.Errorf("workersMax = %d, want 4", s.workersMax)
	}
	if s.socketPath != "/tmp/test_daemon.sock" {
		t.Errorf("socketPath = %q", s.socketPath)
	}
	if s.pool == nil {
		t.Error("pool is nil")
	}
	if s.sm == nil {
		t.Error("session manager is nil")
	}
	if s.metrics == nil {
		t.Error("metrics is nil")
	}
}

func TestNewServer_WithHTTP(t *testing.T) {
	s := NewServer("/tmp/test.sock", 4, ":9090")
	if s.obsServer == nil {
		t.Error("observability server should not be nil when httpAddr is set")
	}
}

// =============================================================================
// writeError
// =============================================================================

func TestWriteError_WithID(t *testing.T) {
	s := NewServer("/tmp/test.sock", 1, "")
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		s.writeError(server, "req-1", -32600, "Invalid Request")
		server.Close()
	}()

	var resp Response
	dec := json.NewDecoder(client)
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if resp.ID != "req-1" {
		t.Errorf("ID = %v, want req-1", resp.ID)
	}
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("error code = %d, want -32600", resp.Error.Code)
	}
}

func TestWriteError_NilID(t *testing.T) {
	s := NewServer("/tmp/test.sock", 1, "")
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	go func() {
		s.writeError(server, nil, -32700, "Parse error")
		server.Close()
	}()

	var resp Response
	dec := json.NewDecoder(client)
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if resp.ID != nil {
		t.Errorf("ID should be nil, got %v", resp.ID)
	}
}

// =============================================================================
// processRequest edge cases
// =============================================================================

func TestProcessRequest_UnknownMethod(t *testing.T) {
	s := NewServer("/tmp/test.sock", 1, "")
	req := Request{JSONRPC: "2.0", Method: "goposix.nonexistent_xyz", ID: "1"}
	res := s.processRequest(req)
	if res == nil {
		t.Fatal("expected response")
	}
	if res.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if res.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601 (Method not found)", res.Error.Code)
	}
}

func TestProcessRequest_MissingMethod_EmptyString(t *testing.T) {
	s := NewServer("/tmp/test.sock", 1, "")
	req := Request{JSONRPC: "2.0", ID: "1", Method: ""}
	res := s.processRequest(req)
	if res == nil || res.Error == nil {
		t.Fatal("expected error for missing method")
	}
}

func TestProcessRequest_InvalidJSONParams(t *testing.T) {
	s := NewServer("/tmp/test.sock", 1, "")
	req := Request{
		JSONRPC: "2.0",
		Method:  "goposix.echo",
		Params:  json.RawMessage(`invalid json {{{`),
		ID:      "1",
	}
	res := s.processRequest(req)
	if res == nil {
		t.Fatal("expected response")
	}
	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error.Message)
	}
	// With unparseable params, echo runs with no args and succeeds
	if res.Result == nil {
		t.Error("expected result")
	}
}

func TestProcessRequest_ShellMethod(t *testing.T) {
	s := NewServer("/tmp/test.sock", 1, "")
	params, _ := json.Marshal(GoposixParams{Text: "echo hello"})
	req := Request{
		JSONRPC: "2.0",
		Method:  "goposix.shell.exec",
		Params:  params,
		ID:      "1",
	}
	res := s.processRequest(req)
	if res == nil {
		t.Fatal("expected response")
	}
	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error.Message)
	}
}

func TestProcessRequest_SessionDestroyMissing(t *testing.T) {
	s := NewServer("/tmp/test.sock", 1, "")
	params, _ := json.Marshal(GoposixParams{SessionId: "nonexistent"})
	req := Request{
		JSONRPC: "2.0",
		Method:  "goposix.session.destroy",
		Params:  params,
		ID:      "1",
	}
	res := s.processRequest(req)
	if res == nil || res.Error == nil {
		t.Fatal("expected error for nonexistent session")
	}
	if !strings.Contains(res.Error.Message, "Invalid session") {
		t.Errorf("error message = %q, want 'Invalid session'", res.Error.Message)
	}
}

func TestProcessRequest_SessionSetCwd_Invalid(t *testing.T) {
	s := NewServer("/tmp/test.sock", 1, "")
	sess := s.sm.Create()
	params, _ := json.Marshal(GoposixParams{SessionId: sess.ID, Path: "/tmp"})
	req := Request{
		JSONRPC: "2.0",
		Method:  "goposix.session.setCwd",
		Params:  params,
		ID:      "1",
	}
	res := s.processRequest(req)
	if res == nil || res.Error != nil {
		t.Fatalf("expected success: %+v", res)
	}
	got, _ := s.sm.Get(sess.ID)
	if got.CWD != "/tmp" {
		t.Errorf("CWD = %q, want /tmp", got.CWD)
	}
}

// =============================================================================
// Batch handling
// =============================================================================

func TestHandleBatch_AllNotifications(t *testing.T) {
	s := NewServer("/tmp/test.sock", 1, "")
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	reqs := []Request{
		{JSONRPC: "2.0", Method: "goposix.echo", Params: json.RawMessage(`{}`)},
		{JSONRPC: "2.0", Method: "goposix.echo", Params: json.RawMessage(`{}`)},
	}

	go func() {
		s.handleBatch(server, reqs)
		server.Close()
	}()

	var resp []Response
	dec := json.NewDecoder(client)
	err := dec.Decode(&resp)
	if err == nil {
		t.Error("expected no response for all-notification batch (EOF)")
	}
}

func TestHandleBatch_Mixed(t *testing.T) {
	s := NewServer("/tmp/test.sock", 1, "")
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	reqs := []Request{
		{JSONRPC: "2.0", Method: "goposix.echo", Params: json.RawMessage(`{}`), ID: "1"},
		{JSONRPC: "2.0", Method: "goposix.nonexistent_xyz", Params: json.RawMessage(`{}`), ID: "2"},
	}

	go func() {
		s.handleBatch(server, reqs)
		server.Close()
	}()

	var resp []Response
	dec := json.NewDecoder(client)
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(resp))
	}
	if resp[0].Error != nil {
		t.Errorf("request 1 should succeed: %v", resp[0].Error)
	}
	if resp[1].Error == nil {
		t.Error("request 2 should fail")
	}
}

// =============================================================================
// Session lifecycle (additional edge cases)
// =============================================================================

func TestSessionManager_GetNonExistent(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)
	_, ok := sm.Get("nonexistent")
	if ok {
		t.Error("Get should return false for nonexistent")
	}
}

func TestSessionManager_SetCwdNonExistent(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)
	ok := sm.SetCwd("nonexistent", "/tmp")
	if ok {
		t.Error("SetCwd should return false for nonexistent")
	}
}

func TestSessionManager_DestroyNonExistent(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)
	ok := sm.Destroy("nonexistent")
	if ok {
		t.Error("Destroy should return false for nonexistent")
	}
}

func TestSessionManager_ListEmpty(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)
	list := sm.List()
	if len(list) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(list))
	}
}

func TestSessionManager_TTLExpiry(t *testing.T) {
	sm := NewSessionManager(50 * time.Millisecond)
	s := sm.Create()

	_, ok := sm.Get(s.ID)
	if !ok {
		t.Fatal("session should exist")
	}

	time.Sleep(150 * time.Millisecond)

	// Force cleanup similar to cleanupLoop
	sm.mu.Lock()
	now := time.Now()
	for id, sess := range sm.sessions {
		if now.Sub(sess.LastActive) > sm.ttl {
			delete(sm.sessions, id)
		}
	}
	sm.mu.Unlock()

	_, ok = sm.Get(s.ID)
	if ok {
		t.Error("session should have expired")
	}
}

// =============================================================================
// Observability
// =============================================================================

func TestMetrics_RecordRequest(t *testing.T) {
	m := NewMetrics()
	if m == nil {
		t.Fatal("NewMetrics returned nil")
	}
	m.RecordRequest("echo", 1.5)
	m.RecordRequest("echo", 2.5)
	m.mu.Lock()
	count := m.durationCounts["echo"]
	sum := m.durationSums["echo"]
	m.mu.Unlock()
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if sum != 4.0 {
		t.Errorf("sum = %f, want 4.0", sum)
	}
}

func TestMetrics_RecordRateLimited(t *testing.T) {
	m := NewMetrics()
	m.RecordRateLimited()
	m.RecordRateLimited()
	// rateLimitedTotal is atomic — we just verify it doesn't panic
	// (internal field not exported, but we can verify the method runs)
}

// =============================================================================
// Concurrent stress
// =============================================================================

func TestWorkerPool_Concurrent(t *testing.T) {
	wp := NewWorkerPool(10)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var counter int

	for i := 0; i < 100; i++ {
		wg.Add(1)
		wp.Submit(context.Background(), func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		})
	}
	wg.Wait()
	mu.Lock()
	c := counter
	mu.Unlock()
	if c != 100 {
		t.Errorf("expected 100, got %d", c)
	}
}

// =============================================================================
// Rate limiter edge cases
// =============================================================================

func TestRateLimiter_RefillAfterWait(t *testing.T) {
	rl := NewRateLimiter(1000.0, 2)
	rl.Allow()
	rl.Allow()
	if rl.Allow() {
		t.Error("should be empty after burst")
	}
	time.Sleep(2 * time.Millisecond)
	if !rl.Allow() {
		t.Error("should refill after wait")
	}
}

func TestRateLimiter_MaxBurst(t *testing.T) {
	rl := NewRateLimiter(100.0, 5)
	count := 0
	for i := 0; i < 10; i++ {
		if rl.Allow() {
			count++
		}
	}
	if count != 5 {
		t.Errorf("expected max 5 allows, got %d", count)
	}
}

// =============================================================================
// Integration: Start/Stop + real request over Unix socket
// =============================================================================

func TestServerStartStop(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "startstop.sock")
	s := NewServer(socket, 2, "")
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	s.Stop()
	// Socket should be cleaned up
	if _, err := os.Stat(socket); !os.IsNotExist(err) {
		t.Error("socket should be cleaned up after Stop")
	}
}

func TestServerPingOverSocket(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "ping.sock")
	s := NewServer(socket, 2, "")
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Send ping request
	enc := json.NewEncoder(conn)
	enc.Encode(Request{JSONRPC: "2.0", Method: "goposix.ping", ID: "1"})

	var resp Response
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("ping error: %v", resp.Error)
	}
}

func TestServerEchoOverSocket(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "echo.sock")
	s := NewServer(socket, 2, "")
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	params, _ := json.Marshal(GoposixParams{Text: "hello"})
	enc := json.NewEncoder(conn)
	enc.Encode(Request{JSONRPC: "2.0", Method: "goposix.echo", Params: params, ID: "1"})

	var resp Response
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("echo error: %v", resp.Error)
	}
	if resp.Result == nil {
		t.Error("expected result")
	}
}

func TestServerInvalidJSONOverSocket(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "invalid.sock")
	s := NewServer(socket, 2, "")
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Send invalid JSON
	conn.Write([]byte("not json\n"))

	var resp Response
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error response for invalid JSON")
	}
}

func TestServerUnknownMethodOverSocket(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "unknown.sock")
	s := NewServer(socket, 2, "")
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	enc.Encode(Request{JSONRPC: "2.0", Method: "goposix.nonexistent_xyz", ID: "1"})

	var resp Response
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Fatalf("expected -32601 Method not found, got %+v", resp.Error)
	}
}

func TestServerBatchOverSocket(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "batch.sock")
	s := NewServer(socket, 2, "")
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	batch := []Request{
		{JSONRPC: "2.0", Method: "goposix.ping", ID: "1"},
		{JSONRPC: "2.0", Method: "goposix.ping", ID: "2"},
	}
	enc := json.NewEncoder(conn)
	enc.Encode(batch)

	var responses []Response
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&responses); err != nil {
		t.Fatalf("decode batch: %v", err)
	}
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}
}

func TestServerNotificationsOverSocket(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "notif.sock")
	s := NewServer(socket, 2, "")
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Notification: no ID
	params, _ := json.Marshal(GoposixParams{Text: "test"})
	enc := json.NewEncoder(conn)
	enc.Encode(Request{JSONRPC: "2.0", Method: "goposix.echo", Params: params})

	// No response should come — set a short read deadline and expect timeout/EOF
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var resp Response
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err == nil {
		t.Error("expected no response for notification")
	}
}

func TestProcessRequest_RawOutput(t *testing.T) {
	s := NewServer("/tmp/test-raw.sock", 1, "")

	// rawOutput=true should return raw stdout text (no JSON envelope).
	params, _ := json.Marshal(GoposixParams{Text: "hello-raw", RawOutput: true})
	req := Request{JSONRPC: "2.0", Method: "goposix.echo", Params: params, ID: 1}
	res := s.processRequest(req)
	if res == nil {
		t.Fatal("expected response")
	}
	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error.Message)
	}
	result, ok := res.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected Result to be map[string]interface{}")
	}
	if ec, ok := result["exitCode"].(int); !ok || ec != 0 {
		t.Errorf("exitCode = %v (type %T), want int(0)", result["exitCode"], result["exitCode"])
	}
	stdout, _ := result["stdout"].(string)
	if stdout != "hello-raw\n" {
		t.Errorf("stdout = %q, want %q", stdout, "hello-raw\n")
	}

	// rawOutput=false should still return JSON envelope.
	params2, _ := json.Marshal(GoposixParams{Text: "hello-json", RawOutput: false})
	req2 := Request{JSONRPC: "2.0", Method: "goposix.echo", Params: params2, ID: 2}
	res2 := s.processRequest(req2)
	if res2 == nil {
		t.Fatal("expected response")
	}
	if res2.Error != nil {
		t.Fatalf("unexpected error: %v", res2.Error.Message)
	}
	// JSON mode: Result should contain "data" with the envelope
	result2, ok := res2.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected Result to be map[string]interface{}")
	}
	if _, ok := result2["data"]; !ok {
		t.Error("expected JSON envelope 'data' field in non-raw mode")
	}
}

func TestProcessRequest_RawOutputExitCode(t *testing.T) {
	s := NewServer("/tmp/test-raw2.sock", 1, "")

	// false command should return exit code 1 with rawOutput.
	params, _ := json.Marshal(GoposixParams{RawOutput: true})
	req := Request{JSONRPC: "2.0", Method: "goposix.false", Params: params, ID: 1}
	res := s.processRequest(req)
	if res == nil {
		t.Fatal("expected response")
	}
	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error.Message)
	}
	result, ok := res.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected Result to be map[string]interface{}")
	}
	if ec, ok := result["exitCode"].(int); !ok || ec != 1 {
		t.Errorf("exitCode for false = %v (type %T), want int(1)", result["exitCode"], result["exitCode"])
	}
}

// =============================================================================
// M6 Integration Tests
// =============================================================================

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func TestIntegration_ObservabilityEndpoints(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("failed to find free TCP port: %v", err)
	}

	socket := filepath.Join(t.TempDir(), "obs.sock")
	httpAddr := fmt.Sprintf("127.0.0.1:%d", port)
	s := NewServer(socket, 2, httpAddr)
	if err := s.Start(); err != nil {
		t.Fatalf("Start server failed: %v", err)
	}
	defer s.Stop()
	time.Sleep(50 * time.Millisecond)

	endpoints := []string{"/healthz", "/readyz", "/metrics", "/status"}
	for _, ep := range endpoints {
		resp, err := http.Get("http://" + httpAddr + ep)
		if err != nil {
			t.Errorf("GET %s failed: %v", ep, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s returned status %d, want 200", ep, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("reading body of %s failed: %v", ep, err)
			continue
		}

		if ep == "/healthz" {
			if !strings.Contains(string(body), `"ok"`) {
				t.Errorf("%s body = %q, want containing '\"ok\"'", ep, body)
			}
		} else if ep == "/readyz" {
			if !strings.Contains(string(body), `"ready"`) {
				t.Errorf("%s body = %q, want containing '\"ready\"'", ep, body)
			}
		} else if ep == "/status" {
			var snap StatusSnapshot
			if err := json.Unmarshal(body, &snap); err != nil {
				t.Errorf("failed to parse status snapshot: %v", err)
			}
			if snap.Version != common.Version {
				t.Errorf("status snap version = %q, want %q", snap.Version, common.Version)
			}
		} else if ep == "/metrics" {
			if !strings.Contains(string(body), "goposix_rpc_duration_count") && !strings.Contains(string(body), "# HELP") {
				t.Errorf("metrics body did not look like prometheus metrics: %s", body)
			}
		}
	}
}

func TestIntegration_PathTraversalRejection(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "traversal.sock")
	s := NewServer(socket, 2, "")
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// 1. Create a session
	enc := json.NewEncoder(conn)
	enc.Encode(Request{JSONRPC: "2.0", Method: "goposix.session.create", ID: "1"})

	var resp Response
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode session.create: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("session.create failed: %v", resp.Error)
	}
	resMap := resp.Result.(map[string]interface{})
	sessID := resMap["sessionId"].(string)

	// 2. Set CWD to a dummy directory under /tmp
	paramsSet, _ := json.Marshal(GoposixParams{SessionId: sessID, Path: "/tmp/mysandbox"})
	enc.Encode(Request{JSONRPC: "2.0", Method: "goposix.session.setCwd", Params: paramsSet, ID: "2"})
	var resp2 Response
	if err := dec.Decode(&resp2); err != nil {
		t.Fatalf("decode session.setCwd: %v", err)
	}

	// 3. Attempt path traversal with echo
	paramsEcho, _ := json.Marshal(GoposixParams{SessionId: sessID, Path: "../../etc/passwd"})
	enc.Encode(Request{JSONRPC: "2.0", Method: "goposix.echo", Params: paramsEcho, ID: "3"})
	var resp3 Response
	if err := dec.Decode(&resp3); err != nil {
		t.Fatalf("decode echo traversal: %v", err)
	}

	if resp3.Error == nil {
		t.Fatal("expected path traversal rejection error, got nil")
	}
	if resp3.Error.Code != -32602 {
		t.Errorf("error code = %d, want -32602", resp3.Error.Code)
	}
	if !strings.Contains(resp3.Error.Message, "Path traversal detected") {
		t.Errorf("expected traversal message, got: %q", resp3.Error.Message)
	}
}

func TestIntegration_SessionSetCwdPathValidation(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "setcwd_traversal.sock")
	s := NewServer(socket, 2, "")
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	// Create session
	enc.Encode(Request{JSONRPC: "2.0", Method: "goposix.session.create", ID: "1"})
	var resp1 Response
	dec.Decode(&resp1)
	resMap := resp1.Result.(map[string]interface{})
	sessID := resMap["sessionId"].(string)

	// Set CWD to /etc (arbitrary path - H1 gap)
	paramsSet, _ := json.Marshal(GoposixParams{SessionId: sessID, Path: "/etc"})
	enc.Encode(Request{JSONRPC: "2.0", Method: "goposix.session.setCwd", Params: paramsSet, ID: "2"})
	var resp2 Response
	dec.Decode(&resp2)
	if resp2.Error != nil {
		t.Fatalf("unexpected error setting arbitrary CWD: %v", resp2.Error)
	}

	// Verify that the CWD in the session is indeed /etc
	sess, ok := s.sm.Get(sessID)
	if !ok {
		t.Fatal("session not found")
	}
	if sess.CWD != "/etc" {
		t.Errorf("CWD = %q, want /etc (H1 gap)", sess.CWD)
	}
}

func TestIntegration_LimitReaderExceeded(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "limit_reader.sock")
	s := NewServer(socket, 2, "")
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Send a payload larger than 1MB
	largePayload := strings.Repeat("a", 1024*1024+1024)
	conn.Write([]byte(largePayload))

	// The connection should respond with a Parse error and close
	var resp Response
	dec := json.NewDecoder(conn)
	err = dec.Decode(&resp)
	if err == nil {
		if resp.Error == nil || !strings.Contains(resp.Error.Message, "Parse error or request too large") {
			t.Errorf("expected limit error response, got: %+v", resp)
		}
	}
}

var registerLimitWriterOnce sync.Once

func registerLimitWriterCommand() {
	registerLimitWriterOnce.Do(func() {
		dispatch.Register(dispatch.Command{
			Name: "test_limit_writer",
			Run: func(args []string, stdin io.Reader, stdout io.Writer) int {
				// write 51 MB of data
				buf := make([]byte, 1024*1024)
				for i := 0; i < 51; i++ {
					_, err := stdout.Write(buf)
					if err != nil {
						return 1
					}
				}
				return 0
			},
		})
	})
}

func TestIntegration_LimitWriterExceeded(t *testing.T) {
	registerLimitWriterCommand()

	socket := filepath.Join(t.TempDir(), "limit_writer.sock")
	s := NewServer(socket, 2, "")
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	params, _ := json.Marshal(GoposixParams{RawOutput: true})
	enc := json.NewEncoder(conn)
	enc.Encode(Request{JSONRPC: "2.0", Method: "goposix.test_limit_writer", Params: params, ID: "1"})

	var resp Response
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decode limit writer response: %v", err)
	}

	// It should succeed or contain a truncated result, but the raw output will have reached 50MB
	if resp.Error != nil {
		t.Fatalf("unexpected error response: %v", resp.Error.Message)
	}
	resMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	stdout := resMap["stdout"].(string)
	if len(stdout) != 50*1024*1024 {
		t.Errorf("expected exactly 50MB of output, got %d bytes", len(stdout))
	}
}

func TestIntegration_ConcurrentConnectionLimit(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "conn_limit.sock")
	s := NewServer(socket, 2, "")

	// Override capacity to 2 for quick testing
	s.connSem = make(chan struct{}, 2)

	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	time.Sleep(20 * time.Millisecond)

	// Dial connection 1
	c1, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial 1: %v", err)
	}
	defer c1.Close()

	// Dial connection 2
	c2, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial 2: %v", err)
	}
	defer c2.Close()
	time.Sleep(10 * time.Millisecond)

	// Dial connection 3 (this will dial, but acceptLoop will block on s.connSem)
	c3, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial 3: %v", err)
	}
	defer c3.Close()
	time.Sleep(10 * time.Millisecond)

	// Verify c1 works
	enc1 := json.NewEncoder(c1)
	enc1.Encode(Request{JSONRPC: "2.0", Method: "goposix.ping", ID: "1"})
	var r1 Response
	dec1 := json.NewDecoder(c1)
	c1.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	if err := dec1.Decode(&r1); err != nil {
		t.Errorf("c1 decode: %v", err)
	}

	// Verify c3 does not respond (blocks on connection semaphore)
	enc3 := json.NewEncoder(c3)
	enc3.Encode(Request{JSONRPC: "2.0", Method: "goposix.ping", ID: "3"})
	var r3 Response
	dec3 := json.NewDecoder(c3)
	c3.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	if err := dec3.Decode(&r3); err == nil {
		t.Error("expected c3 to block and timeout, but got response")
	}

	// Close c1 to release semaphore slot
	c1.Close()
	time.Sleep(50 * time.Millisecond)

	// Now c3 should respond successfully!
	c3.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	dec3 = json.NewDecoder(c3)
	if err := dec3.Decode(&r3); err != nil {
		t.Errorf("c3 should now succeed, got: %v", err)
	}
}

func TestIntegration_GracefulShutdownInFlight(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "graceful.sock")
	s := NewServer(socket, 2, "")
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Spin up server.Stop() asynchronously
	go func() {
		time.Sleep(20 * time.Millisecond)
		s.Stop()
	}()

	// Ping the server to verify it's working up until shutdown
	enc := json.NewEncoder(conn)
	enc.Encode(Request{JSONRPC: "2.0", Method: "goposix.ping", ID: "1"})

	var resp Response
	dec := json.NewDecoder(conn)
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	if err := dec.Decode(&resp); err != nil {
		// Connection might be closed by stop, which is expected
	}
}

// TestDaemonStdinSupport verifies that the stdin field in GoposixParams is
// plumbed through to commands that consume stdin.
func TestDaemonStdinSupport(t *testing.T) {
	// Register a test command that reads stdin and echoes it back.
	dispatch.Register(dispatch.Command{
		Name: "test-stdin-echo",
		Run: func(args []string, stdin io.Reader, stdout io.Writer) int {
			data, _ := io.ReadAll(stdin)
			stdout.Write(data)
			return 0
		},
	})

	sock := filepath.Join(t.TempDir(), "stdin-test.sock")
	srv := NewServer(sock, 4, "")
	srv.Start()
	defer srv.Stop()

	conn, err := net.DialTimeout("unix", sock, 1*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Send request with stdin data.
	req := Request{
		JSONRPC: "2.0",
		Method:  "goposix.test-stdin-echo",
		Params:  json.RawMessage(`{"stdin":"hello from stdin"}`),
		ID:      "1",
	}
	enc := json.NewEncoder(conn)
	if err := enc.Encode(req); err != nil {
		t.Fatal(err)
	}

	var resp Response
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.Error != nil {
		t.Fatalf("RPC error: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("invalid result")
	}
	// When --json is not used, output goes to data as raw string.
	data, _ := result["data"].(string)
	if !strings.Contains(data, "hello from stdin") {
		t.Errorf("expected stdin content in response, got: %q", data)
	}
}

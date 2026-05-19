package daemon

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
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

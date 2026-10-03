package daemon

import (
	"bytes"
	"encoding/json"
	"net"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewServerEnvOverrides(t *testing.T) {
	t.Setenv("GOPOSIX_RATE_LIMIT", "250")
	t.Setenv("GOPOSIX_MAX_REQUEST_SIZE", "4096")
	s := NewServer("/tmp/none.sock", 2, "")
	if s.rateLimit != 250 {
		t.Errorf("rateLimit = %v, want 250", s.rateLimit)
	}
	if s.requestLimit != 4096 {
		t.Errorf("requestLimit = %d, want 4096", s.requestLimit)
	}
}

func TestNewServerEnvInvalidValues(t *testing.T) {
	t.Setenv("GOPOSIX_RATE_LIMIT", "not-a-number")
	t.Setenv("GOPOSIX_MAX_REQUEST_SIZE", "xyz")
	s := NewServer("/tmp/none.sock", 2, "")
	if s.rateLimit != 100 {
		t.Errorf("rateLimit = %v, want default 100", s.rateLimit)
	}
	if s.requestLimit != 1024*1024 {
		t.Errorf("requestLimit = %d, want default 1MB", s.requestLimit)
	}
}

func TestStartInvalidSocketDir(t *testing.T) {
	s := NewServer("/proc/definitely-not-here/goposix.sock", 2, "")
	if err := s.Start(); err == nil {
		s.Stop()
		t.Fatal("expected Start to fail on non-creatable directory")
	}
}

func TestStopWithoutStart(t *testing.T) {
	t.Setenv("GOPOSIX_SHUTDOWN_TIMEOUT", "200ms")
	s := NewServer(filepath.Join(t.TempDir(), "s.sock"), 2, "")
	s.Stop() // must not panic or hang
}

func TestStopForceClosesActiveConnection(t *testing.T) {
	t.Setenv("GOPOSIX_SHUTDOWN_TIMEOUT", "100ms")
	s := NewServer(filepath.Join(t.TempDir(), "s.sock"), 2, "")
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("unix", s.socketPath)
	if err != nil {
		s.Stop()
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond) // let handleConn track the connection

	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop hung with an active connection")
	}

	conn.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 1)
	if _, err := conn.Read(buf); err == nil {
		t.Error("expected connection to be closed by shutdown")
	}
	conn.Close()
}

func TestHandleConnGarbageInput(t *testing.T) {
	s, conn := startRawConn(t)
	defer s.Stop()
	defer conn.Close()

	if _, err := conn.Write([]byte("this is not json\n")); err != nil {
		t.Fatal(err)
	}
	resp := readRawResponse(t, conn)
	if !strings.Contains(resp, "Parse error") {
		t.Errorf("response = %s, want parse error", resp)
	}
}

func TestHandleConnEmptyBatch(t *testing.T) {
	s, conn := startRawConn(t)
	defer s.Stop()
	defer conn.Close()

	if _, err := conn.Write([]byte("[]\n")); err != nil {
		t.Fatal(err)
	}
	resp := readRawResponse(t, conn)
	if !strings.Contains(resp, "Invalid Request") {
		t.Errorf("response = %s, want invalid request", resp)
	}
}

func TestHandleConnRateLimitExceeded(t *testing.T) {
	t.Setenv("GOPOSIX_RATE_LIMIT", "1")
	s, conn := startRawConn(t)
	defer s.Stop()
	defer conn.Close()

	// Send two requests back to back; the second must be rate limited.
	req := []byte(`{"jsonrpc":"2.0","id":1,"method":"goposix.ping"}` + "\n")
	if _, err := conn.Write(req); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write(req); err != nil {
		t.Fatal(err)
	}

	var sawResult, sawRateLimit bool
	dec := json.NewDecoder(conn)
	for i := 0; i < 2; i++ {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		var resp map[string]json.RawMessage
		if err := dec.Decode(&resp); err != nil {
			t.Fatalf("decode response %d: %v", i+1, err)
		}
		raw, _ := json.Marshal(resp)
		if bytes.Contains(raw, []byte("Rate limit exceeded")) {
			sawRateLimit = true
		}
		if _, ok := resp["result"]; ok {
			sawResult = true
		}
	}
	if !sawResult {
		t.Error("first request should have succeeded")
	}
	if !sawRateLimit {
		t.Error("second request should have been rate limited")
	}
}

func TestProcTitleString(t *testing.T) {
	s := NewServer("/tmp/none.sock", 4, "")
	atomic.StoreInt32(&s.activeWorkers, 3)
	atomic.StoreInt64(&s.totalRequests, 7)
	if got := s.procTitle(); got != "goposix daemon [W:3/4 S:0 C:7]" {
		t.Errorf("procTitle = %q", got)
	}
}

func startRawConn(t *testing.T) (*Server, net.Conn) {
	t.Helper()
	s := NewServer(filepath.Join(t.TempDir(), "s.sock"), 2, "")
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("unix", s.socketPath)
	if err != nil {
		s.Stop()
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond) // let handleConn pick up the connection
	return s, conn
}

func readRawResponse(t *testing.T, conn net.Conn) string {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && n == 0 {
		t.Fatalf("read response: %v", err)
	}
	return string(buf[:n])
}

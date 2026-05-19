// Package daemon implements the persistent JSON-RPC 2.0 server.
package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/internal/shell"
	"github.com/ramayac/goposix/pkg/common"
	"log/slog"
)

// Request is a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

// Error represents a JSON-RPC 2.0 error object.
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// WorkerPool provides a bounded concurrency pool.
type WorkerPool struct {
	sem chan struct{}
}

// NewWorkerPool creates a new pool.
func NewWorkerPool(size int) *WorkerPool {
	return &WorkerPool{sem: make(chan struct{}, size)}
}

// Submit queues a function to run.
func (wp *WorkerPool) Submit(ctx context.Context, fn func()) error {
	select {
	case wp.sem <- struct{}{}:
		go func() {
			defer func() { <-wp.sem }()
			fn()
		}()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Server is the daemon server.
type Server struct {
	socketPath string
	listener   net.Listener
	pool       *WorkerPool
	uptime     time.Time

	sm *SessionManager

	activeWorkers int32
	totalRequests int64

	connSem      chan struct{}
	connWG       sync.WaitGroup
	shuttingDown int32

	workersMax int
	metrics    *Metrics
	obsServer  *ObservabilityServer
}

// NewServer creates a new daemon server.
func NewServer(socketPath string, workers int, httpAddr string) *Server {
	m := NewMetrics()
	s := &Server{
		socketPath: socketPath,
		pool:       NewWorkerPool(workers),
		uptime:     time.Now(),
		sm:         NewSessionManager(30 * time.Minute),
		connSem:    make(chan struct{}, 100), // Max 100 concurrent connections
		workersMax: workers,
		metrics:    m,
	}

	if httpAddr != "" {
		s.obsServer = NewObservabilityServer(httpAddr, &s.totalRequests, &s.activeWorkers,
			workers, s.uptime, &s.shuttingDown, s.sm, m)
	}

	return s
}

// Start begins listening on the unix socket.
func (s *Server) Start() error {
	// Remove stale socket
	os.Remove(s.socketPath)

	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}
	s.listener = l

	// Set permissions (0660)
	if err := os.Chmod(s.socketPath, 0660); err != nil {
		l.Close()
		return err
	}

	if s.obsServer != nil {
		if err := s.obsServer.Start(); err != nil {
			l.Close()
			return err
		}
	}

	go s.acceptLoop()
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() {
	atomic.StoreInt32(&s.shuttingDown, 1)
	if s.obsServer != nil {
		s.obsServer.Stop()
	}
	if s.listener != nil {
		s.listener.Close()
	}
	s.connWG.Wait()
	os.Remove(s.socketPath)
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if atomic.LoadInt32(&s.shuttingDown) == 1 {
				return
			}
			// If listener closed, stop accepting
			select {
			case <-time.After(10 * time.Millisecond):
				if s.listener == nil {
					return
				}
			}
			continue
		}

		s.connSem <- struct{}{}
		s.connWG.Add(1)

		go func(c net.Conn) {
			defer s.connWG.Done()
			defer func() { <-s.connSem }()
			s.handleConn(c)
		}(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	rl := NewRateLimiter(100000.0, 100000)
	limitReader := io.LimitReader(conn, 1024*1024)

	// Parse incoming JSON stream. It can be a single object or array.
	dec := json.NewDecoder(limitReader)
	for {
		// Read raw message to check if it's array or object
		var raw json.RawMessage
		err := dec.Decode(&raw)
		if err != nil {
			if err != io.EOF {
				s.writeError(conn, nil, -32700, "Parse error or request too large")
			}
			return
		}

		if !rl.Allow() {
			s.metrics.RecordRateLimited()
			s.writeError(conn, nil, -32000, "Rate limit exceeded")
			continue
		}

		if len(raw) > 0 && raw[0] == '[' {
			// Batch request
			var reqs []Request
			if err := json.Unmarshal(raw, &reqs); err != nil {
				s.writeError(conn, nil, -32600, "Invalid Request")
				continue
			}
			
			if len(reqs) == 0 {
				s.writeError(conn, nil, -32600, "Invalid Request")
				continue
			}

			s.handleBatch(conn, reqs)
		} else {
			// Single request
			var req Request
			if err := json.Unmarshal(raw, &req); err != nil {
				s.writeError(conn, nil, -32600, "Invalid Request")
				continue
			}
			s.handleSingleAsync(conn, req)
		}
	}
}

func (s *Server) handleBatch(conn net.Conn, reqs []Request) {
	var wg sync.WaitGroup
	results := make([]*Response, len(reqs))

	for i, req := range reqs {
		wg.Add(1)
		idx := i
		r := req
		err := s.pool.Submit(context.Background(), func() {
			defer wg.Done()
			res := s.processRequest(r)
			if res != nil {
				results[idx] = res
			}
		})
		if err != nil {
			wg.Done()
			results[idx] = &Response{
				JSONRPC: "2.0",
				ID:      r.ID,
				Error:   &Error{Code: -32000, Message: "Server busy"},
			}
		}
	}
	wg.Wait()

	// Remove nil responses (notifications)
	var final []Response
	for _, res := range results {
		if res != nil {
			final = append(final, *res)
		}
	}

	if len(final) > 0 {
		enc := json.NewEncoder(conn)
		enc.Encode(final)
	}
}

func (s *Server) handleSingleAsync(conn net.Conn, req Request) {
	done := make(chan struct{})
	err := s.pool.Submit(context.Background(), func() {
		defer close(done)
		res := s.processRequest(req)
		if res != nil {
			enc := json.NewEncoder(conn)
			enc.Encode(res)
		}
	})
	if err != nil {
		if req.ID != nil {
			s.writeError(conn, req.ID, -32000, "Server busy")
		}
		return
	}
	// Wait for worker to finish writing the response before returning
	// to the caller (handleConn). This prevents the caller from calling
	// dec.Decode() on the connection while a worker is still holding
	// a pool semaphore slot, which would deadlock under concurrent load.
	<-done
}

func (s *Server) writeError(conn net.Conn, id interface{}, code int, msg string) {
	enc := json.NewEncoder(conn)
	enc.Encode(Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &Error{Code: code, Message: msg},
	})
}

// GoposixParams is used to parse the standard parameters.
type GoposixParams struct {
	Flags     []string `json:"flags"`
	Path      string   `json:"path"`
	Text      string   `json:"text"`
	SessionId string   `json:"sessionId"`
	RawOutput bool     `json:"rawOutput"` // skip --json, return raw stdout text
}

func (s *Server) processRequest(req Request) *Response {
	start := time.Now()

	var (
		rpcExitCode int
		rpcError    string
		rpcCmd      string
	)

	atomic.AddInt64(&s.totalRequests, 1)
	atomic.AddInt32(&s.activeWorkers, 1)
	defer atomic.AddInt32(&s.activeWorkers, -1)

	defer func() {
		duration := time.Since(start)
		durationMs := float64(duration.Microseconds()) / 1000.0
		sessionId := ""
		if len(req.Params) > 0 {
			var p GoposixParams
			if err := json.Unmarshal(req.Params, &p); err == nil {
				sessionId = p.SessionId
			}
		}
		args := []any{
			"method", req.Method,
			"sessionId", sessionId,
			"durationMs", durationMs,
		}
		if rpcCmd != "" {
			args = append(args, "cmd", rpcCmd)
		}
		if rpcExitCode != 0 {
			args = append(args, "exitCode", rpcExitCode)
		}
		if rpcError != "" {
			args = append(args, "error", rpcError)
		}
		slog.Info("rpc handled", args...)
		if s.metrics != nil {
			s.metrics.RecordRequest(req.Method, durationMs)
		}
	}()

	if req.JSONRPC != "2.0" {
		rpcError = "Invalid Request"
		if req.ID != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: -32600, Message: "Invalid Request"}}
		}
		return nil
	}

	if len(req.Method) > 256 {
		rpcError = "Method too long"
		if req.ID != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: -32600, Message: "Method too long"}}
		}
		return nil
	}

	if req.Method == "goposix.ping" {
		if req.ID == nil {
		rpcCmd = "ping"
			return nil
		}
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"pong":    true,
				"uptime":  time.Since(s.uptime).String(),
				"version": common.Version,
				"workers": map[string]interface{}{
					"active": atomic.LoadInt32(&s.activeWorkers),
				},
			},
		}
	}

	if req.Method == "goposix.session.create" {
		rpcCmd = "session.create"
		if req.ID == nil { return nil }
		s := s.sm.Create()
		return &Response{JSONRPC: "2.0", ID: req.ID, Result: s}
	}

	if req.Method == "goposix.session.setCwd" {
		rpcCmd = "session.setCwd"
		if req.ID == nil { return nil }
		var p struct {
			SessionId string `json:"sessionId"`
			Path      string `json:"path"`
		}
		if err := json.Unmarshal(req.Params, &p); err == nil {
			if s.sm.SetCwd(p.SessionId, p.Path) {
				return &Response{JSONRPC: "2.0", ID: req.ID, Result: true}
			}
		}
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: -32602, Message: "Invalid session"}}
	}

	if req.Method == "goposix.session.list" {
		rpcCmd = "session.list"
		if req.ID == nil { return nil }
		sessions := s.sm.List()
		return &Response{JSONRPC: "2.0", ID: req.ID, Result: sessions}
	}

	if req.Method == "goposix.session.destroy" {
		rpcCmd = "session.destroy"
		if req.ID == nil { return nil }
		var p struct {
			SessionId string `json:"sessionId"`
		}
		if err := json.Unmarshal(req.Params, &p); err == nil {
			if s.sm.Destroy(p.SessionId) {
				return &Response{JSONRPC: "2.0", ID: req.ID, Result: true}
			}
		}
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: -32602, Message: "Invalid session"}}
	}

	if req.Method == "goposix.shell.exec" {
		rpcCmd = "shell.exec"
		if req.ID == nil { return nil }
		var p struct {
			SessionId string `json:"sessionId"`
			Script    string `json:"script"`
		}
		if err := json.Unmarshal(req.Params, &p); err == nil {
			cwd := "/"
			var env map[string]string
			if p.SessionId != "" {
				if session, ok := s.sm.Get(p.SessionId); ok {
					cwd = session.CWD
					env = session.Env
				}
			}
			res := shell.Exec(p.Script, cwd, env)
			return &Response{JSONRPC: "2.0", ID: req.ID, Result: res}
		}
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: -32602, Message: "Invalid params"}}
	}

	if !strings.HasPrefix(req.Method, "goposix.") {
		if req.ID != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: -32601, Message: "Method not found"}}
		}
		return nil
	}

	cmdName := strings.TrimPrefix(req.Method, "goposix.")
		rpcCmd = cmdName
	cmd, ok := dispatch.Lookup(cmdName)
	if !ok {
		if req.ID != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: -32601, Message: "Method not found"}}
		}
		return nil
	}

	// Build args
	var args []string
	var session *Session
	var rawOutput bool

	if len(req.Params) > 0 {
		var p GoposixParams
		var dynMap map[string]interface{}
		
		if err := json.Unmarshal(req.Params, &p); err == nil {
			rawOutput = p.RawOutput
			if p.SessionId != "" {
				session, _ = s.sm.Get(p.SessionId)
			}

			if p.Path != "" {
				base := "/"
				if session != nil && session.CWD != "" {
					base = session.CWD
				}
				securePath, err := common.SecurePath(p.Path, base)
				if err != nil {
					rpcError = "Path traversal detected"
					if req.ID != nil {
						return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: -32602, Message: "Path traversal detected"}}
					}
					return nil
				}
				p.Path = securePath
			}

			args = append(args, p.Flags...)
			if cmdName == "echo" && p.Text != "" {
				args = append(args, p.Text)
			}
			if p.Path != "" {
				args = append(args, p.Path)
			}
		} else if err := json.Unmarshal(req.Params, &dynMap); err == nil {
		}
	}

	// Prepend --json so utilities with custom flag parsers (echo) see it first.
	// common.ParseFlags handles --json at any position; echo's parseEchoFlags only
	// scans flags at the start of the argument list.
	// Skip --json when rawOutput is requested (CLI forwarding preserves human-readable output).
	if !rawOutput {
		args = append([]string{"--json"}, args...)
	}

	var buf bytes.Buffer
	// 50MB response limit to prevent OOM
	lw := &common.LimitWriter{W: &buf, Limit: 50 * 1024 * 1024}

	// Execute the command
	exitCode := cmd.Run(args, lw)
		rpcExitCode = exitCode

	// rawOutput: return the raw stdout text directly (used by CLI forwarder).
	if rawOutput {
		if req.ID == nil {
			return nil
		}
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"exitCode": exitCode,
				"stdout":   buf.String(),
			},
		}
	}

	// We intercept the output which should be a JSONEnvelope.
	// But `buf` might contain multiple lines or other things if the utility misbehaves.
	// We only want the JSON output to embed in our result.
	var env common.JSONEnvelope
	
	// Try parsing it
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		// It might be raw ndjson or something else, or completely empty (true/false)
		if req.ID != nil {
			var resultData interface{} = nil
			if buf.Len() > 0 {
				resultData = buf.String()
			}
			return &Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"exitCode": exitCode,
					"data":     resultData,
				},
			}
		}
		return nil
	}

	if req.ID == nil {
		return nil
	}

	// If there's an envelope error, map it
	if env.Error != nil {
		rpcError = env.Error.Message
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    -32000,
				Message: env.Error.Message,
				Data: map[string]interface{}{
					"exitCode": exitCode,
					"code":     env.Error.Code,
				},
			},
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"exitCode": exitCode,
			"data":     env.Data,
		},
	}
}

// RunDaemon sets up signal handling and runs until terminated.
func RunDaemon(socketPath string, workers int, httpAddr string) error {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	slog.Info("starting goposix daemon", "socket", socketPath, "workers", workers)

	if os.Getenv("GOPOSIX_DEBUG") == "1" {
		go func() {
			slog.Info("starting pprof on :6060")
			if err := http.ListenAndServe("localhost:6060", nil); err != nil {
				slog.Error("pprof server failed", "error", err)
			}
		}()
	}

	server := NewServer(socketPath, workers, httpAddr)
	if err := server.Start(); err != nil {
		slog.Error("failed to start server", "error", err)
		return err
	}

	// PID file
	pidFile := "/var/run/goposix.pid"
	if socketPath != "/var/run/goposix.sock" {
		pidFile = socketPath + ".pid" // mostly for testing in /tmp
	}
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c // wait for signal

	// Graceful shutdown
	server.Stop()
	os.Remove(pidFile)
	return nil
}

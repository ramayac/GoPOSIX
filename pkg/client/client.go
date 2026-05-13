// Package client provides a Go JSON-RPC 2.0 client for the korego daemon.
//
// Features:
//   - Connection pooling with configurable pool size
//   - Batch request support
//   - Retry with exponential backoff on transient errors
//   - Context propagation on every call
//   - Typed helper methods for all korego utilities
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"
)

// Client is a JSON-RPC 2.0 client connected to a korego daemon.
type Client struct {
	socketPath string
	pool       *connPool
	timeout    time.Duration
	maxRetries int

	nextID int
	mu     sync.Mutex
}

// Option configures a Client.
type Option func(*Client)

// WithPoolSize sets the maximum number of concurrent connections.
func WithPoolSize(n int) Option {
	return func(c *Client) {
		if n < 1 {
			n = 1
		}
		c.pool.maxSize = n
	}
}

// WithTimeout sets the per-call deadline.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// WithMaxRetries sets the maximum retry count on transient errors.
func WithMaxRetries(n int) Option {
	return func(c *Client) { c.maxRetries = n }
}

// New creates a new Client connected to the daemon at socketPath.
func New(socketPath string, opts ...Option) (*Client, error) {
	c := &Client{
		socketPath: socketPath,
		timeout:    30 * time.Second,
		maxRetries: 2,
		pool: &connPool{
			maxSize: 4,
		},
	}
	for _, o := range opts {
		o(c)
	}
	c.pool.sem = make(chan struct{}, c.pool.maxSize)
	return c, nil
}

// Dial is a convenience constructor kept for backward compatibility.
// Prefer New with options for new code.
func Dial(socketPath string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	c, _ := New(socketPath, WithTimeout(timeout), WithPoolSize(1))
	return c
}

// Close closes all idle connections in the pool.
func (c *Client) Close() error {
	c.pool.mu.Lock()
	defer c.pool.mu.Unlock()
	for _, conn := range c.pool.conns {
		conn.Close()
	}
	c.pool.conns = nil
	return nil
}

// nextID returns a monotonically increasing request ID.
func (c *Client) getNextID() int {
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	c.mu.Unlock()
	return id
}

// --- connection pool ---

type connPool struct {
	mu      sync.Mutex
	conns   []net.Conn
	maxSize int
	sem     chan struct{}
}

func (p *connPool) get(ctx context.Context) (net.Conn, error) {
	select {
	case p.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	p.mu.Lock()
	if len(p.conns) > 0 {
		conn := p.conns[len(p.conns)-1]
		p.conns = p.conns[:len(p.conns)-1]
		p.mu.Unlock()
		return conn, nil
	}
	p.mu.Unlock()

	return nil, nil // nil means caller must dial
}

func (p *connPool) put(conn net.Conn) {
	if conn == nil {
		<-p.sem
		return
	}
	p.mu.Lock()
	p.conns = append(p.conns, conn)
	p.mu.Unlock()
	<-p.sem
}

func (p *connPool) discard(conn net.Conn) {
	if conn != nil {
		conn.Close()
	}
	<-p.sem
}

// --- core RPC ---

// rpcRequest is a JSON-RPC 2.0 request.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// rpcResponse is a JSON-RPC 2.0 response.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

// Call executes a single JSON-RPC call with retry support.
// If result is non-nil, the response result is unmarshaled into it.
func (c *Client) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	id := c.getNextID()

	rawParams, err := marshalOptional(params)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}

	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  rawParams,
		ID:      id,
	}

	var resp rpcResponse
	err = c.do(ctx, req, &resp)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return resp.Error
	}

	if result != nil && resp.Result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}
	return nil
}

// CallRaw is like Call but returns the raw result and error for custom handling.
func (c *Client) CallRaw(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := c.getNextID()

	rawParams, err := marshalOptional(params)
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}

	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  rawParams,
		ID:      id,
	}

	var resp rpcResponse
	err = c.do(ctx, req, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp.Result, nil
}

// Notify sends a JSON-RPC notification (no response expected).
func (c *Client) Notify(ctx context.Context, method string, params interface{}) error {
	rawParams, err := marshalOptional(params)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}

	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  rawParams,
		// no ID = notification
	}

	// Notifications get no response, so we just need to send.
	return c.doSendOnly(ctx, req)
}

// BatchRequest is a single request within a batch.
type BatchRequest struct {
	Method string
	Params interface{}
}

// BatchResponse is a single response from a batch call.
type BatchResponse struct {
	Result json.RawMessage
	Error  *rpcError
}

// Batch executes multiple requests in one round-trip. Results are returned in
// the same order as requests.
func (c *Client) Batch(ctx context.Context, reqs []BatchRequest) ([]BatchResponse, error) {
	if len(reqs) == 0 {
		return nil, errors.New("batch: empty request list")
	}

	rpcReqs := make([]rpcRequest, len(reqs))
	for i, r := range reqs {
		rawParams, err := marshalOptional(r.Params)
		if err != nil {
			return nil, fmt.Errorf("batch[%d] marshal params: %w", i, err)
		}
		rpcReqs[i] = rpcRequest{
			JSONRPC: "2.0",
			Method:  r.Method,
			Params:  rawParams,
			ID:      c.getNextID(),
		}
	}

	// Batch is not retried — partial success/failure is ambiguous.
	var rpcResps []rpcResponse
	if err := c.doSingle(ctx, rpcReqs, &rpcResps); err != nil {
		return nil, err
	}

	results := make([]BatchResponse, len(rpcResps))
	for i, resp := range rpcResps {
		results[i] = BatchResponse{Result: resp.Result, Error: resp.Error}
	}
	return results, nil
}

// --- retry / transport ---

var (
	errShuttingDown = errors.New("daemon is shutting down")
)

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errShuttingDown) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	// Connection refused / broken pipe / EOF are retryable
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	// Check for syscall errors via string (portable enough)
	msg := err.Error()
	return msg == "connection refused" || msg == "broken pipe"
}

func (c *Client) do(ctx context.Context, req interface{}, resp interface{}) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 100 * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
		lastErr = c.doSingle(ctx, req, resp)
		if lastErr == nil {
			return nil
		}
		if !isRetryable(lastErr) {
			return lastErr
		}
	}
	return lastErr
}

func (c *Client) doSingle(ctx context.Context, req interface{}, resp interface{}) error {
	conn, err := c.pool.get(ctx)
	if err != nil {
		return err
	}

	if conn == nil {
		dialer := net.Dialer{}
		conn, err = dialer.DialContext(ctx, "unix", c.socketPath)
		if err != nil {
			c.pool.discard(nil)
			return err
		}
	}

	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	} else {
		conn.SetDeadline(time.Now().Add(c.timeout))
	}

	enc := json.NewEncoder(conn)
	if err := enc.Encode(req); err != nil {
		c.pool.discard(conn)
		return err
	}

	dec := json.NewDecoder(conn)
	if err := dec.Decode(resp); err != nil {
		c.pool.discard(conn)
		if errors.Is(err, io.EOF) {
			return io.EOF
		}
		return err
	}

	c.pool.put(conn)
	return nil
}

func (c *Client) doSendOnly(ctx context.Context, req interface{}) error {
	conn, err := c.pool.get(ctx)
	if err != nil {
		return err
	}

	if conn == nil {
		dialer := net.Dialer{}
		conn, err = dialer.DialContext(ctx, "unix", c.socketPath)
		if err != nil {
			c.pool.discard(nil)
			return err
		}
	}

	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	} else {
		conn.SetDeadline(time.Now().Add(c.timeout))
	}

	enc := json.NewEncoder(conn)
	if err := enc.Encode(req); err != nil {
		c.pool.discard(conn)
		return err
	}

	c.pool.put(conn)
	return nil
}

func marshalOptional(v interface{}) (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

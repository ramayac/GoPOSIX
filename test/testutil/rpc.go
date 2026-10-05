// Package testutil provides a minimal JSON-RPC client for tests.
//
// It mirrors the wire protocol of the goposix daemon. It has no connection
// pooling, retry, or batch support. Tests do not need those features.
package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// Client is a minimal JSON-RPC client for a goposix daemon.
//
// It uses one connection. Call is safe for concurrent use: a mutex serializes
// requests so each response matches its request.
type Client struct {
	mu      sync.Mutex
	conn    net.Conn
	enc     *json.Encoder
	dec     *json.Decoder
	timeout time.Duration
	dialErr error
	nextID  int
}

// Dial connects to the daemon at the unix socket path.
//
// It returns a Client even when the dial fails. The error is returned by the
// first Call so test code can stay simple: c := testutil.Dial(socket, timeout).
func Dial(socketPath string, timeout time.Duration) *Client {
	c := &Client{timeout: timeout}
	conn, err := net.DialTimeout("unix", socketPath, timeout)
	if err != nil {
		c.dialErr = err
		return c
	}
	c.conn = conn
	c.enc = json.NewEncoder(conn)
	c.dec = json.NewDecoder(conn)
	return c
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Call sends one JSON-RPC request and decodes the response into result.
func (c *Client) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	if c.dialErr != nil {
		return c.dialErr
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	req := struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params,omitempty"`
		ID      int             `json:"id"`
	}{JSONRPC: "2.0", Method: method, ID: c.nextID}

	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal params: %w", err)
		}
		req.Params = b
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(c.timeout)
	}
	if err := c.conn.SetDeadline(deadline); err != nil {
		return err
	}

	if err := c.enc.Encode(req); err != nil {
		return err
	}

	var resp struct {
		JSONRPC string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result,omitempty"`
		Error   *struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data,omitempty"`
		} `json:"error,omitempty"`
		ID interface{} `json:"id"`
	}
	if err := c.dec.Decode(&resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	if result != nil && resp.Result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}
	return nil
}

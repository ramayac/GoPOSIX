package posixjson_test

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ramayac/korego/internal/daemon"
	"github.com/ramayac/korego/pkg/client"
	_ "github.com/ramayac/korego/pkg/cat"
	_ "github.com/ramayac/korego/pkg/echo"
	_ "github.com/ramayac/korego/pkg/sed"
	_ "github.com/ramayac/korego/pkg/sleep"
	_ "github.com/ramayac/korego/pkg/tee"
	_ "github.com/ramayac/korego/pkg/testcmd"
	_ "github.com/ramayac/korego/pkg/tr"
	_ "github.com/ramayac/korego/pkg/truefalse"
	_ "github.com/ramayac/korego/pkg/yes"
)

// ResultWrapper represents the standardized output structure for KoreGo JSON-RPC
type ResultWrapper struct {
	ExitCode int         `json:"exitCode"`
	Data     interface{} `json:"data"`
}

func startDaemon(t *testing.T) string {
	socketPath := filepath.Join(t.TempDir(), "korego.sock")
	
	// Start daemon in background
	go func() {
		err := daemon.RunDaemon(socketPath, 2, "")
		if err != nil {
			t.Logf("daemon exited: %v", err)
		}
	}()

	// Wait for socket to be created
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			return socketPath
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("daemon socket not created in time")
	return ""
}

func TestStructuredOutputSemantics(t *testing.T) {
	socket := startDaemon(t)
	c := client.Dial(socket, 5*time.Second)

	tests := []struct {
		name       string
		method     string
		params     interface{}
		expectCode int
		verifyData func(t *testing.T, data interface{})
	}{
		{
			name:       "true utility exits 0 with value=true",
			method:     "korego.true",
			params:     nil,
			expectCode: 0,
			verifyData: func(t *testing.T, data interface{}) {
				m, ok := data.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map, got %T", data)
				}
				if v := m["value"]; v != true {
					t.Errorf("expected value=true, got %v", v)
				}
				if ec := m["exitCode"]; ec != float64(0) {
					t.Errorf("expected exitCode=0, got %v", ec)
				}
			},
		},
		{
			name:       "false utility exits 1 with value=false",
			method:     "korego.false",
			params:     nil,
			expectCode: 1,
			verifyData: func(t *testing.T, data interface{}) {
				m, ok := data.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map, got %T", data)
				}
				if v := m["value"]; v != false {
					t.Errorf("expected value=false, got %v", v)
				}
				if ec := m["exitCode"]; ec != float64(1) {
					t.Errorf("expected exitCode=1, got %v", ec)
				}
			},
		},
		{
			name:       "echo utility returns text and exits 0",
			method:     "korego.echo",
			params:     map[string]interface{}{"text": "hello posix"},
			expectCode: 0,
			verifyData: func(t *testing.T, data interface{}) {
				m, ok := data.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map[string]interface{}, got %T", data)
				}
				if text := m["text"]; text != "hello posix" {
					t.Errorf("expected text 'hello posix', got '%v'", text)
				}
			},
		},
		{
			name:       "sleep utility returns duration info",
			method:     "korego.sleep",
			params:     map[string]interface{}{"flags": []interface{}{"0.001"}},
			expectCode: 0,
			verifyData: func(t *testing.T, data interface{}) {
				m, ok := data.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map, got %T", data)
				}
				if d, ok := m["duration"]; !ok || d.(float64) <= 0 {
					t.Errorf("expected duration > 0, got %v", d)
				}
				if r, ok := m["requested"]; !ok || r.(float64) <= 0 {
					t.Errorf("expected requested > 0, got %v", r)
				}
			},
		},
		{
			name:       "yes utility returns string/count in json mode",
			method:     "korego.yes",
			params:     nil,
			expectCode: 0,
			verifyData: func(t *testing.T, data interface{}) {
				m, ok := data.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map, got %T", data)
				}
				if s := m["string"]; s != "y" {
					t.Errorf("expected string='y', got %v", s)
				}
				if c := m["count"]; c.(float64) != 1 {
					t.Errorf("expected count=1, got %v", c)
				}
				if tr := m["truncated"]; tr != true {
					t.Errorf("expected truncated=true, got %v", tr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result ResultWrapper
			err := c.Call(context.Background(), tt.method, tt.params, &result)
			
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			if result.ExitCode != tt.expectCode {
				t.Errorf("expected exit code %d, got %d", tt.expectCode, result.ExitCode)
			}
			
			if tt.verifyData != nil {
				tt.verifyData(t, result.Data)
			}
		})
	}
	
	t.Run("cat utility fails on missing file and preserves exit code", func(t *testing.T) {
		// Manual call to inspect the Error object
		conn, err := net.Dial("unix", socket)
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		defer conn.Close()
		
		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "korego.cat",
			"params":  map[string]interface{}{"path": "/does/not/exist/ever"},
			"id":      1,
		}
		b, _ := json.Marshal(req)
		conn.Write(b)
		
		var res map[string]interface{}
		dec := json.NewDecoder(conn)
		if err := dec.Decode(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		
		if res["error"] != nil {
			// If it eventually uses RenderError, it'll be here
			errObj := res["error"].(map[string]interface{})
			dataObj := errObj["data"].(map[string]interface{})
			if int(dataObj["exitCode"].(float64)) != 1 {
				t.Errorf("expected exit code 1 in error, got %v", dataObj["exitCode"])
			}
			return
		}
		
		resObj := res["result"].(map[string]interface{})
		if int(resObj["exitCode"].(float64)) != 1 {
			t.Errorf("expected exit code 1 in result, got %v", resObj["exitCode"])
		}
	})
	
	t.Run("test utility via daemon returns bool result", func(t *testing.T) {
		conn, err := net.Dial("unix", socket)
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		defer conn.Close()

		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "korego.test",
			"params":  map[string]interface{}{"flags": []interface{}{"hello", "=", "hello"}},
			"id":      2,
		}
		b, _ := json.Marshal(req)
		conn.Write(b)

		var res map[string]interface{}
		dec := json.NewDecoder(conn)
		if err := dec.Decode(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		resObj := res["result"].(map[string]interface{})
		if int(resObj["exitCode"].(float64)) != 0 {
			t.Errorf("expected exit code 0, got %v", resObj["exitCode"])
		}
		data := resObj["data"].(map[string]interface{})
		if data["result"] != true {
			t.Errorf("expected result=true, got %v", data["result"])
		}
	})
}

package posixjson_test

import (
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
	_ "github.com/ramayac/korego/pkg/truefalse"
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
			name:       "true utility exits 0 with nil data",
			method:     "korego.true",
			params:     nil,
			expectCode: 0,
			verifyData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Errorf("expected nil data for true, got %v", data)
				}
			},
		},
		{
			name:       "false utility exits 1 with nil data",
			method:     "korego.false",
			params:     nil,
			expectCode: 1,
			verifyData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Errorf("expected nil data for false, got %v", data)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result ResultWrapper
			err := c.Call(tt.method, tt.params, &result)
			
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
}

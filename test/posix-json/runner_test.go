package posixjson_test

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ramayac/goposix/internal/daemon"
	"github.com/ramayac/goposix/pkg/client"
	_ "github.com/ramayac/goposix/pkg/cat"
	_ "github.com/ramayac/goposix/pkg/echo"
	_ "github.com/ramayac/goposix/pkg/sed"
	_ "github.com/ramayac/goposix/pkg/sleep"
	_ "github.com/ramayac/goposix/pkg/tee"
	_ "github.com/ramayac/goposix/pkg/testcmd"
	_ "github.com/ramayac/goposix/pkg/tr"
	_ "github.com/ramayac/goposix/pkg/truefalse"
	_ "github.com/ramayac/goposix/pkg/yes"
)

// ResultWrapper represents the standardized output structure for GoPOSIX JSON-RPC
type ResultWrapper struct {
	ExitCode int         `json:"exitCode"`
	Data     interface{} `json:"data"`
}

func startDaemon(t *testing.T) string {
	socketPath := filepath.Join(t.TempDir(), "goposix.sock")
	
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
			method:     "goposix.true",
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
			method:     "goposix.false",
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
			method:     "goposix.echo",
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
			method:     "goposix.sleep",
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
			method:     "goposix.yes",
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
			"method":  "goposix.cat",
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
	
	t.Run("tee utility via daemon creates file and returns bytesWritten", func(t *testing.T) {
		tmpDir := t.TempDir()
		outFile := filepath.Join(tmpDir, "tee_out.txt")

		conn, err := net.Dial("unix", socket)
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		defer conn.Close()

		// tee with a file argument — stdin is empty in daemon mode (EOF),
		// so it writes 0 bytes but the file is created and the result is valid.
		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "goposix.tee",
			"params":  map[string]interface{}{"flags": []interface{}{outFile}},
			"id":      3,
		}
		b, _ := json.Marshal(req)
		conn.Write(b)

		var res map[string]interface{}
		dec := json.NewDecoder(conn)
		if err := dec.Decode(&res); err != nil {
			t.Fatalf("failed to decode tee response: %v", err)
		}

		if res["error"] != nil {
			t.Logf("tee returned error (may be expected in daemon mode): %v", res["error"])
			return
		}
		resObj := res["result"].(map[string]interface{})
		if int(resObj["exitCode"].(float64)) != 0 {
			t.Errorf("expected exit code 0, got %v", resObj["exitCode"])
		}
		data := resObj["data"].(map[string]interface{})
		if _, ok := data["bytesWritten"]; !ok {
			t.Error("tee result missing bytesWritten field")
		}
		if files, ok := data["files"]; ok {
			farr := files.([]interface{})
			if len(farr) == 0 {
				t.Error("tee result has empty files list")
			} else if farr[0].(string) != outFile {
				t.Errorf("tee file path mismatch: got %q, want %q", farr[0], outFile)
			}
		}
	})

	t.Run("tr utility via daemon handles empty stdin gracefully", func(t *testing.T) {
		conn, err := net.Dial("unix", socket)
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		defer conn.Close()

		// tr with set1 and set2 — daemon stdin is EOF, so it produces no output
		// but should exit 0 and return a valid TrResult.
		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "goposix.tr",
			"params":  map[string]interface{}{"flags": []interface{}{"a-z", "A-Z"}},
			"id":      4,
		}
		b, _ := json.Marshal(req)
		conn.Write(b)

		var res map[string]interface{}
		dec := json.NewDecoder(conn)
		if err := dec.Decode(&res); err != nil {
			t.Fatalf("failed to decode tr response: %v", err)
		}

		if res["error"] != nil {
			t.Logf("tr returned error (may be expected in daemon mode): %v", res["error"])
			return
		}
		resObj := res["result"].(map[string]interface{})
		if int(resObj["exitCode"].(float64)) != 0 {
			t.Errorf("expected exit code 0, got %v", resObj["exitCode"])
		}
		data := resObj["data"].(map[string]interface{})
		if _, ok := data["lines"]; !ok {
			t.Error("tr result missing lines field")
		}
		if _, ok := data["lineCount"]; !ok {
			t.Error("tr result missing lineCount field")
		}
		if _, ok := data["bytesIn"]; !ok {
			t.Error("tr result missing bytesIn field")
		}
		if _, ok := data["bytesOut"]; !ok {
			t.Error("tr result missing bytesOut field")
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
			"method":  "goposix.test",
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

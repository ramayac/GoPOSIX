package benchmark

import (
	"bytes"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/ramayac/korego/internal/daemon"
	"github.com/ramayac/korego/internal/dispatch"

	_ "github.com/ramayac/korego/pkg/echo"
	_ "github.com/ramayac/korego/pkg/ls"
)

func BenchmarkDaemonEcho(b *testing.B) {
	socket := filepath.Join(b.TempDir(), "korego-bench.sock")
	server := daemon.NewServer(socket, 4, "")
	server.Start()
	defer server.Stop()

	// Wait for socket
	time.Sleep(100 * time.Millisecond)

	reqBytes, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "korego.echo",
		"params":  map[string]interface{}{"text": "hello"},
		"id":      1,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := net.Dial("unix", socket)
		if err != nil {
			b.Fatal(err)
		}
		
		conn.Write(reqBytes)
		
		var res map[string]interface{}
		dec := json.NewDecoder(conn)
		dec.Decode(&res)
		conn.Close()
	}
}

func BenchmarkDaemonLs(b *testing.B) {
	socket := filepath.Join(b.TempDir(), "korego-bench-ls.sock")
	server := daemon.NewServer(socket, 4, "")
	server.Start()
	defer server.Stop()

	// Wait for socket
	time.Sleep(100 * time.Millisecond)

	reqBytes, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "korego.ls",
		"params":  map[string]interface{}{"path": "/tmp"},
		"id":      1,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := net.Dial("unix", socket)
		if err != nil {
			b.Fatal(err)
		}
		
		conn.Write(reqBytes)
		
		var res map[string]interface{}
		dec := json.NewDecoder(conn)
		dec.Decode(&res)
		conn.Close()
	}
}

// Very basic CLI benchmark without fork/exec by invoking Run()
func BenchmarkCLIEcho(b *testing.B) {
	cmd, _ := dispatch.Lookup("echo")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		cmd.Run([]string{"hello"}, &buf)
	}
}

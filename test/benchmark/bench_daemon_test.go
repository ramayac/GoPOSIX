package benchmark

import (
	"bytes"
	"encoding/json"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ramayac/goposix/internal/daemon"
	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/tr"
	"github.com/ramayac/goposix/pkg/wc"

	_ "github.com/ramayac/goposix/pkg/echo"
	_ "github.com/ramayac/goposix/pkg/ls"
)

func BenchmarkDaemonEcho(b *testing.B) {
	socket := filepath.Join(b.TempDir(), "goposix-bench.sock")
	server := daemon.NewServer(socket, 4, "")
	server.Start()
	defer server.Stop()

	// Wait for socket
	time.Sleep(100 * time.Millisecond)

	reqBytes, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "goposix.echo",
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
	socket := filepath.Join(b.TempDir(), "goposix-bench-ls.sock")
	server := daemon.NewServer(socket, 4, "")
	server.Start()
	defer server.Stop()

	// Wait for socket
	time.Sleep(100 * time.Millisecond)

	reqBytes, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "goposix.ls",
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
		cmd.Run([]string{"hello"}, nil, &buf, &buf, "")
	}
}

func BenchmarkCountProper(b *testing.B) {
	testStr := strings.Repeat("hello world this is a test\nand another line here of text\n", 200)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := strings.NewReader(testStr)
		_, err := wc.CountProper(r)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTrTranslate(b *testing.B) {
	testStr := strings.Repeat("abcdefghijklmnopqrstuvwxyz\n", 400)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := strings.NewReader(testStr)
		var out bytes.Buffer
		err := tr.Run(r, &out, "a-z", "A-Z", false, false, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTrSqueeze(b *testing.B) {
	testStr := strings.Repeat("helloooooo   woooorldddd   this   isss   aaa   testtt\n", 200)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := strings.NewReader(testStr)
		var out bytes.Buffer
		err := tr.Run(r, &out, "a-z", "", false, true, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

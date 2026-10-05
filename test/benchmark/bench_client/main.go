// Command bench_client benchmarks the GoPOSIX daemon using a raw JSON-RPC
// client with a persistent connection. It contrasts with the socat-per-call
// approach used in the shell-based benchmark scripts.
//
// Usage:
//
//	bench_client [flags] <call_count>
//
// Flags:
//
//	-socket    string   daemon socket path (default /var/run/goposix.sock)
//	-op        string   operation: echo, cat, ls, grep, wc, find, stat, whoami, rpc-loop (default echo)
//	-workspace string   workspace dir for rpc-loop (default /tmp/bench/rpc_bench/workspace)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ramayac/goposix/test/testutil"
)

func main() {
	socketPath := flag.String("socket", "/var/run/goposix.sock", "daemon socket path")
	op := flag.String("op", "echo", "operation (echo, cat, ls, grep, wc, find, stat, whoami, rpc-loop)")
	workspace := flag.String("workspace", "/tmp/bench/rpc_bench/workspace", "workspace dir for rpc-loop")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: bench_client [flags] <call_count>\n")
		os.Exit(2)
	}

	count := 0
	fmt.Sscanf(flag.Arg(0), "%d", &count)
	if count < 1 {
		count = 1
	}

	c := testutil.Dial(*socketPath, 5*time.Second)
	defer c.Close()

	ctx := context.Background()
	var result map[string]interface{}

	// call runs one JSON-RPC call and exits on error.
	call := func(method string, params interface{}) {
		if err := c.Call(ctx, method, params, &result); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s: %v\n", method, err)
			os.Exit(1)
		}
	}

	start := time.Now()

	switch *op {
	case "echo":
		for i := 0; i < count; i++ {
			call("goposix.echo", map[string]string{"text": "hello"})
		}
	case "cat":
		for i := 0; i < count; i++ {
			call("goposix.cat", map[string]string{"path": *workspace + "/README.md"})
		}
	case "ls":
		for i := 0; i < count; i++ {
			call("goposix.ls", map[string]interface{}{"path": "/bin", "flags": nil})
		}
	case "grep":
		for i := 0; i < count; i++ {
			call("goposix.grep", map[string]interface{}{
				"flags": []string{"TODO", *workspace + "/README.md"},
			})
		}
	case "wc":
		for i := 0; i < count; i++ {
			call("goposix.wc", map[string]string{"path": *workspace + "/README.md"})
		}
	case "find":
		for i := 0; i < count; i++ {
			call("goposix.find", map[string]interface{}{
				"path":  *workspace,
				"flags": []string{"-name", "*.go"},
			})
		}
	case "stat":
		for i := 0; i < count; i++ {
			call("goposix.stat", map[string]string{"path": "/etc/hostname"})
		}
	case "whoami":
		for i := 0; i < count; i++ {
			call("goposix.whoami", nil)
		}

	// rpc-loop simulates a programmatic task loop.
	// Each iteration: ls -la -> cat README -> grep TODO -> wc -> find *.go
	case "rpc-loop":
		readme := *workspace + "/README.md"
		for i := 0; i < count; i++ {
			call("goposix.ls", map[string]interface{}{"path": *workspace, "flags": []string{"-la"}})
			call("goposix.cat", map[string]string{"path": readme})
			call("goposix.grep", map[string]interface{}{"flags": []string{"TODO", readme}})
			call("goposix.wc", map[string]string{"path": readme})
			call("goposix.find", map[string]interface{}{"path": *workspace, "flags": []string{"-name", "*.go"}})
		}

	default:
		ops := []string{"echo", "cat", "ls", "grep", "wc", "find", "stat", "whoami", "rpc-loop"}
		fmt.Fprintf(os.Stderr, "ERROR: unknown operation: %s\n", *op)
		fmt.Fprintf(os.Stderr, "  Supported: %s\n", strings.Join(ops, ", "))
		os.Exit(2)
	}

	elapsed := time.Since(start).Seconds()

	// Output in bench_run-compatible CSV format:
	// label,sample,wall_sec,user_sec,sys_sec,rss_kb
	fmt.Printf("daemon_rpc_%s_%d,%d,%.6f,0,0,0\n", *op, count, 1, elapsed)
}

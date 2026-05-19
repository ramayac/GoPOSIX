// Command bench_client benchmarks the GoPOSIX daemon using the Go SDK client
// with a persistent connection. Contrasts with the socat-per-call approach
// used in the shell-based benchmark scripts.
//
// Usage:
//
//	bench_client [flags] <call_count>
//
// Flags:
//
//	-socket    string   daemon socket path (default /var/run/goposix.sock)
//	-op       string   operation: echo, cat, ls, grep, wc, find, stat, rpc-loop (default echo)
//	-pool      int      connection pool size (default 1)
//	-workspace string   workspace dir for rpc-loop (default /tmp/bench/rpc_bench/workspace)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ramayac/goposix/pkg/client"
)

func main() {
	socketPath := flag.String("socket", "/var/run/goposix.sock", "daemon socket path")
	op := flag.String("op", "echo", "operation (echo, cat, ls, grep, wc, find, stat, whoami, rpc-loop)")
	poolSize := flag.Int("pool", 1, "connection pool size")
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

	c, err := client.New(*socketPath,
		client.WithPoolSize(*poolSize),
		client.WithTimeout(5*time.Second),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: connect: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	ctx := context.Background()
	start := time.Now()

	switch *op {
	case "echo":
		for i := 0; i < count; i++ {
			_, err := c.Echo(ctx, "hello")
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: echo call %d: %v\n", i, err)
				os.Exit(1)
			}
		}
	case "cat":
		for i := 0; i < count; i++ {
			_, err := c.Cat(ctx, *workspace+"/README.md")
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: cat call %d: %v\n", i, err)
				os.Exit(1)
			}
		}
	case "ls":
		for i := 0; i < count; i++ {
			_, err := c.Ls(ctx, "/bin", nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: ls call %d: %v\n", i, err)
				os.Exit(1)
			}
		}
	case "grep":
		for i := 0; i < count; i++ {
			_, err := c.Grep(ctx, "TODO", []string{*workspace + "/README.md"})
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: grep call %d: %v\n", i, err)
				os.Exit(1)
			}
		}
	case "wc":
		for i := 0; i < count; i++ {
			_, err := c.Wc(ctx, *workspace+"/README.md")
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: wc call %d: %v\n", i, err)
				os.Exit(1)
			}
		}
	case "find":
		for i := 0; i < count; i++ {
			_, err := c.Find(ctx, *workspace, []string{"-name", "*.go"})
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: find call %d: %v\n", i, err)
				os.Exit(1)
			}
		}
	case "stat":
		for i := 0; i < count; i++ {
			_, err := c.Stat(ctx, "/etc/hostname")
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: stat call %d: %v\n", i, err)
				os.Exit(1)
			}
		}
	case "whoami":
		for i := 0; i < count; i++ {
			_, err := c.Whoami(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: whoami call %d: %v\n", i, err)
				os.Exit(1)
			}
		}

	// rpc-loop: simulates a programmatic task loop.
	// Each iteration: ls -la → cat README → grep TODO → wc -l → find *.go
	case "rpc-loop":
		readme := *workspace + "/README.md"
		for i := 0; i < count; i++ {
			if _, err := c.Ls(ctx, *workspace, []string{"-la"}); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: rpc-loop ls iter %d: %v\n", i, err)
				os.Exit(1)
			}
			if _, err := c.Cat(ctx, readme); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: rpc-loop cat iter %d: %v\n", i, err)
				os.Exit(1)
			}
			if _, err := c.Grep(ctx, "TODO", []string{readme}); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: rpc-loop grep iter %d: %v\n", i, err)
				os.Exit(1)
			}
			if _, err := c.Wc(ctx, readme); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: rpc-loop wc iter %d: %v\n", i, err)
				os.Exit(1)
			}
			if _, err := c.Find(ctx, *workspace, []string{"-name", "*.go"}); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: rpc-loop find iter %d: %v\n", i, err)
				os.Exit(1)
			}
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
	fmt.Printf("daemon_sdk_%s_%d,%d,%.6f,0,0,0\n", *op, count, 1, elapsed)
}

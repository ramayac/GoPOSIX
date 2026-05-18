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
//	-socket  string   daemon socket path (default /var/run/goposix.sock)
//	-op     string   operation: echo, cat, ls, grep, wc, find, stat (default echo)
//	-pool    int      connection pool size (default 1)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ramayac/goposix/pkg/client"
)

func main() {
	socketPath := flag.String("socket", "/var/run/goposix.sock", "daemon socket path")
	op := flag.String("op", "echo", "operation (echo, cat, ls, grep, wc, find, stat)")
	poolSize := flag.Int("pool", 1, "connection pool size")
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
			_, err := c.Cat(ctx, "/etc/hostname")
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
			_, err := c.Grep(ctx, "root", []string{"/etc/passwd"})
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: grep call %d: %v\n", i, err)
				os.Exit(1)
			}
		}
	case "wc":
		for i := 0; i < count; i++ {
			_, err := c.Wc(ctx, "/etc/hostname")
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: wc call %d: %v\n", i, err)
				os.Exit(1)
			}
		}
	case "find":
		for i := 0; i < count; i++ {
			_, err := c.Find(ctx, "/bin", []string{"-name", "goposix"})
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
	default:
		fmt.Fprintf(os.Stderr, "ERROR: unknown operation: %s\n", *op)
		fmt.Fprintf(os.Stderr, "  Supported: echo, cat, ls, grep, wc, find, stat, whoami\n")
		os.Exit(2)
	}

	elapsed := time.Since(start).Seconds()

	// Output in bench_run-compatible CSV format:
	// label,sample,wall_sec,user_sec,sys_sec,rss_kb
	// user/sys/rss are 0 since we only measure wall time.
	fmt.Printf("daemon_sdk_%s_%d,%d,%.6f,0,0,0\n", *op, count, 1, elapsed)
}

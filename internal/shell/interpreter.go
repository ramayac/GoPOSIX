package shell

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// execMu serializes shell execution to prevent concurrent os.Chdir() calls
// from clobbering each other. os.Chdir changes process-global state, not
// per-goroutine state, so multiple shell.exec RPC calls cannot safely set
// the working directory concurrently.
//
// TODO: Eliminate os.Chdir entirely by threading a CWD parameter through
// the dispatch.Command.Run signature so every utility resolves paths
// against an explicit directory instead of relying on the process CWD.
// This would remove the need for execMu and allow concurrent shell execs.
var execMu sync.Mutex

type ExecResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode uint8  `json:"exitCode"`
}

func Exec(script string, cwd string, env map[string]string) ExecResult {
	execMu.Lock()
	defer execMu.Unlock()

	// Save and restore the process CWD so that sequential Exec calls
	// (each with their own explicit or session-tracked CWD) do not
	// leak cd side-effects into the daemon process.
	origCwd, _ := os.Getwd()
	defer func() { os.Chdir(origCwd) }()

	var stdout, stderr bytes.Buffer

	// 128MB memory limit per stream
	lStdout := &common.LimitWriter{W: &stdout, Limit: 128 * 1024 * 1024}
	lStderr := &common.LimitWriter{W: &stderr, Limit: 128 * 1024 * 1024}

	parser := syntax.NewParser()
	prog, err := parser.Parse(strings.NewReader(script), "")
	if err != nil {
		return ExecResult{Stderr: err.Error(), ExitCode: 127}
	}

	execHandler := func(ctx context.Context, args []string) error {
		if len(args) == 0 {
			return nil
		}
		cmdName := args[0]
		cmd, ok := dispatch.Lookup(cmdName)
		if !ok {
			// Fall back to system exec for commands not registered in dispatch.
			// In production (FROM scratch), there are no system binaries, so this
			// is a no-op. In testing/debug environments, it allows standard Unix
			// commands to work. SecurePath confinement in openHandler prevents
			// path traversal.
			return interp.DefaultExecHandler(0)(ctx, args)
		}

		hc := interp.HandlerCtx(ctx)
		// Sync the shell's working directory to the host process so dispatch
		// commands (ls, pwd, etc.) see the same directory as cd set.
		if hc.Dir != "" {
			os.Chdir(hc.Dir)
		}
		exitCode := cmd.Run(args[1:], hc.Stdin, hc.Stdout)
		if exitCode != 0 {
			return interp.NewExitStatus(uint8(exitCode))
		}
		return nil
	}

	openHandler := func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
		base := cwd
		if base == "" {
			base, _ = os.Getwd()
		}
		if base == "" {
			base = "/"
		}
		securePath, err := common.SecurePath(path, base)
		if err != nil {
			return nil, &os.PathError{Op: "open", Path: path, Err: err}
		}
		return interp.DefaultOpenHandler()(ctx, securePath, flag, perm)
	}

	opts := []interp.RunnerOption{
		interp.StdIO(nil, lStdout, lStderr),
		interp.ExecHandler(execHandler),
		interp.OpenHandler(openHandler),
	}

	if cwd != "" {
		opts = append(opts, interp.Dir(cwd))
	}

	if env != nil {
		var envList []string
		for k, v := range env {
			envList = append(envList, k+"="+v)
		}
		opts = append(opts, interp.Env(expand.ListEnviron(envList...)))
	}

	runner, err := interp.New(opts...)
	if err != nil {
		return ExecResult{Stderr: err.Error(), ExitCode: 127}
	}

	timeout := 30 * time.Second
	if s := os.Getenv("GOPOSIX_SHELL_TIMEOUT"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			timeout = d
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Save the effective baseline CWD: the explicit cwd we passed in,
	// or the host process CWD if no explicit cwd was given.  We only
	// sync back to the host when a cd actually moved away from this
	// baseline, which avoids accidentally pinning the process CWD to
	// a temporary directory that was passed as the explicit cwd.
	baselineCwd := cwd
	if baselineCwd == "" {
		baselineCwd, _ = os.Getwd()
	}
	err = runner.Run(ctx, prog)

	// Apply any cd changes back to the host process so subsequent
	// Exec calls (e.g., in an interactive REPL) start from the
	// correct working directory. mvdan/sh only updates runner.Dir.
	if runner.Dir != "" && runner.Dir != baselineCwd {
		os.Chdir(runner.Dir)
	}

	exitCode := uint8(0)
	if err != nil {
		if exit, ok := interp.IsExitStatus(err); ok {
			exitCode = exit
		} else {
			exitCode = 1
			stderr.WriteString(err.Error() + "\n")
		}
	}

	return ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

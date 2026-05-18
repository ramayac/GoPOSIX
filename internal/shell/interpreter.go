package shell

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ramayac/goposix/internal/dispatch"
	"github.com/ramayac/goposix/pkg/common"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

type ExecResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode uint8  `json:"exitCode"`
}

func Exec(script string, cwd string, env map[string]string) ExecResult {
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
		exitCode := cmd.Run(args[1:], hc.Stdout)
		if exitCode != 0 {
			return interp.NewExitStatus(uint8(exitCode))
		}
		return nil
	}

	openHandler := func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
		base := "/"
		if cwd != "" {
			base = cwd
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

	err = runner.Run(ctx, prog)
	
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

package tooling

import (
	"context"
	"os/exec"
	"path/filepath"
)

// RunTest runs `go test` in cwd with args (e.g. ./...) and returns output and exit code.
func RunTest(ctx context.Context, cwd string, args []string) (output string, exitCode int, err error) {
	return runGo(ctx, cwd, "test", args)
}

// RunBuild runs `go build` in cwd with args and returns output and exit code.
func RunBuild(ctx context.Context, cwd string, args []string) (output string, exitCode int, err error) {
	return runGo(ctx, cwd, "build", args)
}

func runGo(ctx context.Context, cwd, subcmd string, args []string) (output string, exitCode int, err error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", -1, err
	}
	cmdArgs := append([]string{subcmd}, args...)
	cmd := exec.CommandContext(ctx, "go", cmdArgs...)
	cmd.Dir = abs
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return string(out), exitErr.ExitCode(), nil
		}
		return string(out), -1, runErr
	}
	return string(out), 0, nil
}

package tooling

import (
	"context"
	"os/exec"
	"path/filepath"
)

// Ripgrep runs `rg pattern path [--glob ...]` and returns combined stdout+stderr and exit code.
// Exit code 1 (no matches) is not an error.
func Ripgrep(ctx context.Context, pattern, path string, glob []string) (output string, exitCode int, err error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", -1, err
	}
	args := []string{pattern, abs}
	for _, g := range glob {
		if g != "" {
			args = append(args, "--glob", g)
		}
	}
	cmd := exec.CommandContext(ctx, "rg", args...)
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return string(out), exitErr.ExitCode(), nil
		}
		return string(out), -1, runErr
	}
	return string(out), 0, nil
}

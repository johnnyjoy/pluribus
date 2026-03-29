package tooling

import (
	"context"
	"os/exec"
	"path/filepath"
)

// GitDiff runs `git diff base..head` in repoPath and returns combined stdout+stderr.
func GitDiff(ctx context.Context, repoPath, base, head string) (output string, err error) {
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}
	args := []string{"diff", base + ".." + head}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = abs
	out, err := cmd.CombinedOutput()
	return string(out), err
}

package tooling

import (
	"path/filepath"
	"strings"
)

// Allowlist restricts which commands and working directories the tool API may use.
// When Commands or Dirs is empty, no restriction is applied (allow all).
type Allowlist struct {
	commands map[string]bool
	dirs     []string // absolute paths; request dir must be under one of these
}

// NewAllowlist builds an allowlist from config. Empty slices = allow all.
func NewAllowlist(allowedCommands, allowedDirs []string) *Allowlist {
	commands := make(map[string]bool)
	for _, c := range allowedCommands {
		c = strings.TrimSpace(strings.ToLower(c))
		if c != "" {
			commands[c] = true
		}
	}
	var dirs []string
	for _, d := range allowedDirs {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		abs, err := filepath.Abs(d)
		if err != nil {
			continue
		}
		dirs = append(dirs, filepath.Clean(abs))
	}
	return &Allowlist{commands: commands, dirs: dirs}
}

// CheckCommand returns true if the command is allowed (or allowlist has no command restriction).
func (a *Allowlist) CheckCommand(cmd string) bool {
	if len(a.commands) == 0 {
		return true
	}
	return a.commands[strings.ToLower(strings.TrimSpace(cmd))]
}

// CheckDir returns true if the working directory is under one of the allowed dirs (or no dir restriction).
func (a *Allowlist) CheckDir(dir string) bool {
	if len(a.dirs) == 0 {
		return true
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	abs = filepath.Clean(abs)
	for _, allowed := range a.dirs {
		if abs == allowed || strings.HasPrefix(abs, allowed+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

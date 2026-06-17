// Package buildinfo holds link-time release metadata (set via -ldflags from Makefile/Docker).
package buildinfo

import (
	"fmt"
	"strings"
)

// Set at link time: -X control-plane/internal/buildinfo.Version=… etc.
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

// String returns a single-line version stamp for logs and --version output.
func String() string {
	v := strings.TrimSpace(Version)
	if v == "" {
		v = "dev"
	}
	return fmt.Sprintf("%s commit=%s built=%s", v, GitCommit, BuildTime)
}

// JSONFields returns inspectable fields for build proof artifacts.
func JSONFields() map[string]string {
	return map[string]string{
		"version":    Version,
		"git_commit": GitCommit,
		"build_time": BuildTime,
	}
}

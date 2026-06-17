package testisolation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Metrics counts isolation risks in scanned files.
type Metrics struct {
	ProductionMCPDependencyCount    int `json:"production_mcp_dependency_count"`
	CursorMCPConfigDependencyCount  int `json:"cursor_mcp_config_dependency_count"`
	GlobalBinaryDependencyCount     int `json:"global_binary_dependency_count"`
	UnqualifiedPathBinaryCount      int `json:"unqualified_path_binary_count"`
	ExternalEndpointDependencyCount int `json:"external_endpoint_dependency_count"`
	LocalCodebaseTestPathsCount     int `json:"local_codebase_test_paths_count"`
}

// Finding is one scanned violation or allowlisted match.
type Finding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Rule    string `json:"rule"`
	Snippet string `json:"snippet"`
	Allowed bool   `json:"allowed,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// Report is the isolation audit artifact.
type Report struct {
	ScannedFiles []string  `json:"scanned_files"`
	Findings     []Finding `json:"findings"`
	Metrics      Metrics   `json:"metrics"`
	GatePassed   bool      `json:"gate_passed"`
}

// AllowlistEntry documents a permitted pattern.
type AllowlistEntry struct {
	Pattern string `json:"pattern"`
	Rule    string `json:"rule"`
	Reason  string `json:"reason"`
}

type rule struct {
	name    string
	pattern *regexp.Regexp
}

var isolationRules = []rule{
	{name: "cursor_mcp_config", pattern: regexp.MustCompile(`\.cursor/mcp\.json|mcpServers`)},
	{name: "production_url", pattern: regexp.MustCompile(`https?://(api\.|prod\.|production\.|pluribus\.(io|com)|[^/\s"']+\.pluribus)`)},
	{name: "unqualified_pluribus_mcp", pattern: regexp.MustCompile(`exec\.Command(?:Context)?\([^)]*"pluribus-mcp"`)},
	{name: "global_binary_lookup", pattern: regexp.MustCompile(`exec\.LookPath\(\s*"pluribus-mcp"|which pluribus-mcp`)},
	{name: "external_mcp_env", pattern: regexp.MustCompile(`os\.Getenv\(\s*"(?:MCP_URL|PLURIBUS_URL|PLURIBUS_ENDPOINT)"\s*\)`)},
}

var localPatterns = []*regexp.Regexp{
	regexp.MustCompile(`httptest\.New(Server|Recorder|Request)`),
	regexp.MustCompile(`go run \./cmd/pluribus-mcp`),
	regexp.MustCompile(`go build.*\./cmd/pluribus-mcp`),
	regexp.MustCompile(`NewHTTPHandler\(`),
	regexp.MustCompile(`127\.0\.0\.1|localhost`),
}

// AuditRepo scans test and proof paths under repoRoot.
func AuditRepo(repoRoot string, allowlist []AllowlistEntry) (Report, error) {
	var files []string
	scanDirs := []string{
		filepath.Join(repoRoot, "control-plane"),
		filepath.Join(repoRoot, "scripts"),
		filepath.Join(repoRoot, ".github", "workflows"),
	}
	for _, dir := range scanDirs {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			base := filepath.Base(path)
			if strings.HasSuffix(base, "_test.go") ||
				strings.HasPrefix(base, "proof-") && strings.HasSuffix(base, ".sh") ||
				base == "Makefile" || strings.HasSuffix(base, ".yml") {
				files = append(files, path)
			}
			return nil
		})
	}
	var findings []Finding
	localCount := 0
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		content := string(b)
		for _, lp := range localPatterns {
			if lp.MatchString(content) {
				localCount++
				break
			}
		}
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			for _, r := range isolationRules {
				if !r.pattern.MatchString(line) {
					continue
				}
				allowed, reason := matchAllowlist(allowlist, r.name, f, line)
				findings = append(findings, Finding{
					File: rel(repoRoot, f), Line: i + 1, Rule: r.name,
					Snippet: strings.TrimSpace(line), Allowed: allowed, Reason: reason,
				})
			}
		}
	}
	m := computeMetrics(findings, localCount)
	passed := m.ProductionMCPDependencyCount == 0 &&
		m.CursorMCPConfigDependencyCount == 0 &&
		m.GlobalBinaryDependencyCount == 0 &&
		m.UnqualifiedPathBinaryCount == 0 &&
		m.ExternalEndpointDependencyCount == 0
	return Report{
		ScannedFiles: relAll(repoRoot, files),
		Findings:     findings,
		Metrics:      m,
		GatePassed:   passed,
	}, nil
}

func computeMetrics(findings []Finding, localCount int) Metrics {
	m := Metrics{LocalCodebaseTestPathsCount: localCount}
	for _, f := range findings {
		if f.Allowed {
			continue
		}
		switch f.Rule {
		case "cursor_mcp_config":
			m.CursorMCPConfigDependencyCount++
		case "production_url":
			m.ProductionMCPDependencyCount++
			m.ExternalEndpointDependencyCount++
		case "unqualified_pluribus_mcp":
			m.UnqualifiedPathBinaryCount++
			m.GlobalBinaryDependencyCount++
		case "global_binary_lookup":
			m.GlobalBinaryDependencyCount++
		case "external_mcp_env":
			m.ExternalEndpointDependencyCount++
		}
	}
	return m
}

func matchAllowlist(allowlist []AllowlistEntry, rule, file, line string) (bool, string) {
	for _, a := range allowlist {
		if a.Rule != "" && a.Rule != rule {
			continue
		}
		if a.Pattern != "" {
			if matched, _ := regexp.MatchString(a.Pattern, file+" "+line); matched {
				return true, a.Reason
			}
		}
	}
	// Built-in safe patterns
	if rule == "unqualified_pluribus_mcp" && strings.Contains(line, `go run`) {
		return true, "go run builds from current checkout"
	}
	if rule == "production_url" && (strings.Contains(line, "127.0.0.1") || strings.Contains(line, "localhost")) {
		return true, "local test URL"
	}
	if rule == "external_mcp_env" && strings.Contains(line, "srv.URL") {
		return true, "httptest server URL"
	}
	return false, ""
}

func rel(root, path string) string {
	if r, err := filepath.Rel(root, path); err == nil {
		return r
	}
	return path
}

func relAll(root string, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, rel(root, p))
	}
	return out
}

// WriteReport writes isolation JSON artifact.
func WriteReport(report *Report, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// DefaultAllowlist returns documented safe exceptions.
func DefaultAllowlist() []AllowlistEntry {
	return []AllowlistEntry{
		{Rule: "unqualified_pluribus_mcp", Pattern: `go run \./cmd/pluribus-mcp`, Reason: "integration test builds MCP from checkout module root"},
		{Rule: "production_url", Pattern: `127\.0\.0\.1|localhost`, Reason: "local docker/httptest endpoints only"},
		{Rule: "external_mcp_env", Pattern: `CONTROL_PLANE_URL.*srv\.URL`, Reason: "httptest-bound control plane URL"},
	}
}

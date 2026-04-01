package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_slowPathSection(t *testing.T) {
	// Load example config and verify slow_path is parsed and defaults applied
	var path string
	for _, p := range []string{"configs/config.example.yaml", "control-plane/configs/config.example.yaml"} {
		if _, err := os.Stat(p); err == nil {
			path = p
			break
		}
	}
	if path == "" {
		t.Skip("configs/config.example.yaml not found (run from control-plane or repo root)")
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig returned nil")
	}
	// Example has enabled: false
	if cfg.SlowPath.Enabled != false {
		t.Errorf("SlowPath.Enabled = %v, want false", cfg.SlowPath.Enabled)
	}
	// Defaults applied by applySlowPathDefaults when values are zero; example sets them
	if cfg.SlowPath.HighRiskThreshold <= 0 {
		t.Errorf("SlowPath.HighRiskThreshold = %v, want positive", cfg.SlowPath.HighRiskThreshold)
	}
	if cfg.SlowPath.ExpandConstraintsBy <= 0 {
		t.Errorf("SlowPath.ExpandConstraintsBy = %v, want positive", cfg.SlowPath.ExpandConstraintsBy)
	}
	if cfg.SlowPath.ExpandFailuresBy <= 0 {
		t.Errorf("SlowPath.ExpandFailuresBy = %v, want positive", cfg.SlowPath.ExpandFailuresBy)
	}
	if cfg.SlowPath.ExpandPatternsBy <= 0 {
		t.Errorf("SlowPath.ExpandPatternsBy = %v, want positive", cfg.SlowPath.ExpandPatternsBy)
	}
	if !cfg.SlowPathEnabled() {
		// expected when enabled is false
	}
}

func TestConfig_SlowPathEnabled(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *Config
		expect bool
	}{
		{"nil config", nil, false},
		{"enabled false", &Config{SlowPath: SlowPathConfig{Enabled: false}}, false},
		{"enabled true", &Config{SlowPath: SlowPathConfig{Enabled: true}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.SlowPathEnabled()
			if got != tt.expect {
				t.Errorf("SlowPathEnabled() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestLoadConfig_slowPathDefaultsApplied(t *testing.T) {
	// Use a minimal YAML that omits most slow_path fields to verify defaults
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.yaml")
	// Only slow_path.enabled: true, no other keys
	minimal := `
server:
  bind: ":8123"
postgres:
  dsn: "postgres://localhost/test"
slow_path:
  enabled: true
`
	if err := os.WriteFile(path, []byte(minimal), 0644); err != nil {
		t.Fatalf("write minimal config: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.SlowPathEnabled() {
		t.Error("SlowPathEnabled() = false, want true")
	}
	// applySlowPathDefaults should have set numeric defaults
	if cfg.SlowPath.HighRiskThreshold != 1.0 {
		t.Errorf("HighRiskThreshold = %v, want 1.0", cfg.SlowPath.HighRiskThreshold)
	}
	if cfg.SlowPath.ExpandConstraintsBy != 4 {
		t.Errorf("ExpandConstraintsBy = %v, want 4", cfg.SlowPath.ExpandConstraintsBy)
	}
	if cfg.SlowPath.ExpandFailuresBy != 4 {
		t.Errorf("ExpandFailuresBy = %v, want 4", cfg.SlowPath.ExpandFailuresBy)
	}
	if cfg.SlowPath.ExpandPatternsBy != 2 {
		t.Errorf("ExpandPatternsBy = %v, want 2", cfg.SlowPath.ExpandPatternsBy)
	}
}

// TestLoadConfig_lspSection verifies Task 101: lsp section is parsed when present.
func TestLoadConfig_lspSection(t *testing.T) {
	var path string
	for _, p := range []string{"configs/config.example.yaml", "control-plane/configs/config.example.yaml"} {
		if _, err := os.Stat(p); err == nil {
			path = p
			break
		}
	}
	if path == "" {
		t.Skip("configs/config.example.yaml not found")
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.LSP == nil {
		t.Fatal("LSP section missing or not parsed")
	}
	if cfg.LSP.Enabled {
		t.Errorf("LSP.Enabled = true in example, want false")
	}
	if cfg.LSP.RecallSymbolBoost != 0.5 {
		t.Logf("LSP.RecallSymbolBoost = %v (example 0.5)", cfg.LSP.RecallSymbolBoost)
	}
}

func TestSimilarityConfig_IsEnabled(t *testing.T) {
	var nilCfg *SimilarityConfig
	if nilCfg.IsEnabled() {
		t.Fatal("nil *SimilarityConfig must be disabled")
	}
	var s SimilarityConfig
	if !s.IsEnabled() {
		t.Fatal("omitted Enabled must default to true (advisory ingest)")
	}
	off := false
	s.Enabled = &off
	if s.IsEnabled() {
		t.Fatal("explicit false must disable")
	}
	on := true
	s.Enabled = &on
	if !s.IsEnabled() {
		t.Fatal("explicit true must enable")
	}
}

func TestEnforcementConfig_IsEnabled(t *testing.T) {
	var nilCfg *EnforcementConfig
	if nilCfg.IsEnabled() {
		t.Fatal("nil *EnforcementConfig must be disabled")
	}
	var e EnforcementConfig
	if !e.IsEnabled() {
		t.Fatal("omitted Enabled must default to true (RC1)")
	}
	off := false
	e.Enabled = &off
	if e.IsEnabled() {
		t.Fatal("explicit false must disable")
	}
	on := true
	e.Enabled = &on
	if !e.IsEnabled() {
		t.Fatal("explicit true must enable")
	}
}

func TestLoadConfig_synthesisDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "synth.yaml")
	yaml := `
server:
  bind: ":8123"
postgres:
  dsn: "postgres://localhost/test"
synthesis:
  enabled: false
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Synthesis.Enabled {
		t.Fatal("expected synthesis.enabled false")
	}
	if cfg.Synthesis.TimeoutSeconds != 120 {
		t.Errorf("timeout default = %d, want 120", cfg.Synthesis.TimeoutSeconds)
	}
}

func TestLoadConfig_synthesisEnabledInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	yaml := `
server:
  bind: ":8123"
postgres:
  dsn: "postgres://localhost/test"
synthesis:
  enabled: true
  provider: openai
  model: ""
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for empty model when synthesis enabled")
	}
}

func TestApplyPromotionDefaults_clamps(t *testing.T) {
	p := PromotionConfig{MinEvidenceLinks: -1, MinEvidenceScore: -0.5, MinPromoteConfidence: 2.0, MinPolicyComposite: 2.0, SignalNormDivisor: -1}
	applyPromotionDefaults(&p)
	if p.MinEvidenceLinks != 0 {
		t.Errorf("MinEvidenceLinks = %d", p.MinEvidenceLinks)
	}
	if p.MinEvidenceScore != 0 {
		t.Errorf("MinEvidenceScore = %v", p.MinEvidenceScore)
	}
	if p.MinPromoteConfidence != 1 {
		t.Errorf("MinPromoteConfidence = %v, want 1", p.MinPromoteConfidence)
	}
	if p.MinPolicyComposite != 1 {
		t.Errorf("MinPolicyComposite = %v, want 1", p.MinPolicyComposite)
	}
	if p.SignalNormDivisor != 0 {
		t.Errorf("SignalNormDivisor = %v, want 0 (library default 15 when applying composite)", p.SignalNormDivisor)
	}
	if p.AutoMinSupportCount != 4 {
		t.Errorf("AutoMinSupportCount = %d, want 4", p.AutoMinSupportCount)
	}
	if p.AutoMinSalience != 0.7 {
		t.Errorf("AutoMinSalience = %v, want 0.7", p.AutoMinSalience)
	}
}

func TestLoadConfig_recallSemanticAndRankingDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.yaml")
	minimal := `
server:
  bind: ":8123"
postgres:
  dsn: "postgres://localhost/test"
`
	if err := os.WriteFile(path, []byte(minimal), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Recall.SemanticRetrieval == nil {
		t.Fatal("expected SemanticRetrieval struct after defaults")
	}
	if !cfg.Recall.SemanticRetrieval.RetrievalEnabled() {
		t.Error("expected semantic retrieval enabled when YAML omits enabled (nil => on)")
	}
	if cfg.Recall.Ranking == nil {
		t.Fatal("expected Ranking struct after defaults")
	}
	if cfg.Recall.SemanticRetrieval.MaxSemanticCandidates <= 0 {
		t.Errorf("MaxSemanticCandidates = %d", cfg.Recall.SemanticRetrieval.MaxSemanticCandidates)
	}
}

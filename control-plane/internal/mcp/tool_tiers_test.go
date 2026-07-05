package mcp

import (
	"os"
	"testing"
)

func TestFilterRegistryByTier_coreIncludesMemoryFeedback(t *testing.T) {
	t.Cleanup(func() { SetToolsTier(ToolsTierAll) })
	reg := toolRegistry()
	core := filterRegistryByTier(reg, ToolsTierCore)
	found := false
	for _, spec := range core {
		if spec.Name == "memory_feedback" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("memory_feedback must be in core tier for connect-time curation")
	}
}

func TestFilterRegistryByTier_core(t *testing.T) {
	t.Cleanup(func() { SetToolsTier(ToolsTierAll) })
	reg := toolRegistry()
	core := filterRegistryByTier(reg, ToolsTierCore)
	if len(core) != len(coreToolNames) {
		t.Fatalf("core count %d want %d", len(core), len(coreToolNames))
	}
	for _, spec := range core {
		if _, ok := coreToolNames[spec.Name]; !ok {
			t.Fatalf("unexpected core tool %q", spec.Name)
		}
	}
}

func TestFilterRegistryByTier_standard(t *testing.T) {
	t.Cleanup(func() { SetToolsTier(ToolsTierAll) })
	reg := toolRegistry()
	std := filterRegistryByTier(reg, ToolsTierStandard)
	want := len(coreToolNames) + len(standardExtraToolNames)
	if len(std) != want {
		t.Fatalf("standard count %d want %d", len(std), want)
	}
}

func TestInitToolsTier_envOverridesConfig(t *testing.T) {
	t.Cleanup(func() {
		os.Unsetenv("PLURIBUS_TOOLS")
		SetToolsTier(ToolsTierAll)
	})
	t.Setenv("PLURIBUS_TOOLS", "core")
	InitToolsTier("standard")
	if ActiveToolsTier() != ToolsTierCore {
		t.Fatalf("tier %q want core", ActiveToolsTier())
	}
}

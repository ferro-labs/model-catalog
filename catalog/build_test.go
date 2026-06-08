package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCatalog(t *testing.T) {
	tmpDir := t.TempDir()
	providersDir := filepath.Join(tmpDir, "providers")
	distDir := filepath.Join(tmpDir, "dist")

	// Create openai model YAML
	openaiDir := filepath.Join(providersDir, "openai", "models")
	if err := os.MkdirAll(openaiDir, 0o750); err != nil {
		t.Fatalf("mkdir openai: %v", err)
	}
	openaiEntry := Entry{
		Provider:      "openai",
		ModelID:       "gpt-4o",
		DisplayName:   "GPT-4o",
		Mode:          "chat",
		ContextWindow: 128000,
		Lifecycle:     Lifecycle{Status: "active"},
	}
	openaiYAML, err := WriteModelYAML(openaiEntry)
	if err != nil {
		t.Fatalf("marshal openai: %v", err)
	}
	if err := os.WriteFile(filepath.Join(openaiDir, "gpt-4o.yaml"), openaiYAML, 0o600); err != nil {
		t.Fatalf("write openai yaml: %v", err)
	}

	// Create anthropic model YAML
	anthropicDir := filepath.Join(providersDir, "anthropic", "models")
	if err := os.MkdirAll(anthropicDir, 0o750); err != nil {
		t.Fatalf("mkdir anthropic: %v", err)
	}
	anthropicEntry := Entry{
		Provider:      "anthropic",
		ModelID:       "claude-sonnet-4-5",
		DisplayName:   "Claude Sonnet 4.5",
		Mode:          "chat",
		ContextWindow: 200000,
		Lifecycle:     Lifecycle{Status: "active"},
	}
	anthropicYAML, err := WriteModelYAML(anthropicEntry)
	if err != nil {
		t.Fatalf("marshal anthropic: %v", err)
	}
	if err := os.WriteFile(filepath.Join(anthropicDir, "claude-sonnet-4-5.yaml"), anthropicYAML, 0o600); err != nil {
		t.Fatalf("write anthropic yaml: %v", err)
	}

	// Run build
	if err := Build(providersDir, distDir); err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Verify output
	outputPath := filepath.Join(distDir, "catalog.json")
	data, err := os.ReadFile(filepath.Clean(outputPath))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	var result map[string]Entry
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("parse output JSON: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}

	openai, ok := result["openai/gpt-4o"]
	if !ok {
		t.Fatal("missing key openai/gpt-4o")
	}
	if openai.DisplayName != "GPT-4o" {
		t.Errorf("openai DisplayName = %q, want %q", openai.DisplayName, "GPT-4o")
	}

	anthropic, ok := result["anthropic/claude-sonnet-4-5"]
	if !ok {
		t.Fatal("missing key anthropic/claude-sonnet-4-5")
	}
	if anthropic.DisplayName != "Claude Sonnet 4.5" {
		t.Errorf("anthropic DisplayName = %q, want %q", anthropic.DisplayName, "Claude Sonnet 4.5")
	}
}

func TestBuildCatalogWithVersion(t *testing.T) {
	tmpDir := t.TempDir()
	providersDir := filepath.Join(tmpDir, "providers")
	distDir := filepath.Join(tmpDir, "dist")

	writeTestModel(t, providersDir, "openai", "gpt-4o.yaml", Entry{
		Provider:    "openai",
		ModelID:     "gpt-4o",
		DisplayName: "GPT-4o",
		Mode:        "chat",
		Lifecycle:   Lifecycle{Status: "ga"},
		Tier:        "flagship",
	})

	if err := BuildWithVersion(providersDir, distDir, "v2026.06.08.1"); err != nil {
		t.Fatalf("BuildWithVersion() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(distDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if manifest.Version != "v2026.06.08.1" {
		t.Fatalf("manifest.Version = %q, want %q", manifest.Version, "v2026.06.08.1")
	}
}

func TestBuildCatalogRejectsDuplicateKey(t *testing.T) {
	tmpDir := t.TempDir()
	providersDir := filepath.Join(tmpDir, "providers")
	distDir := filepath.Join(tmpDir, "dist")

	writeTestModel(t, providersDir, "openai", "gpt-4o.yaml", Entry{
		Provider:    "openai",
		ModelID:     "gpt-4o",
		DisplayName: "GPT-4o",
		Mode:        "chat",
		Lifecycle:   Lifecycle{Status: "ga"},
		Tier:        "flagship",
	})
	writeTestModel(t, providersDir, "openai", "gpt-4o-copy.yaml", Entry{
		Provider:    "openai",
		ModelID:     "gpt-4o",
		DisplayName: "GPT-4o Copy",
		Mode:        "chat",
		Lifecycle:   Lifecycle{Status: "ga"},
		Tier:        "flagship",
	})

	err := Build(providersDir, distDir)
	if err == nil {
		t.Fatal("expected duplicate catalog key error")
	}
	if !strings.Contains(err.Error(), "duplicate catalog key") {
		t.Fatalf("error = %v, want duplicate catalog key", err)
	}
}

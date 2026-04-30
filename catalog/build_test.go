package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildCatalog(t *testing.T) {
	tmpDir := t.TempDir()
	providersDir := filepath.Join(tmpDir, "providers")
	distDir := filepath.Join(tmpDir, "dist")

	// Create openai model YAML
	openaiDir := filepath.Join(providersDir, "openai", "models")
	if err := os.MkdirAll(openaiDir, 0o755); err != nil {
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
	if err := os.WriteFile(filepath.Join(openaiDir, "gpt-4o.yaml"), openaiYAML, 0o644); err != nil {
		t.Fatalf("write openai yaml: %v", err)
	}

	// Create anthropic model YAML
	anthropicDir := filepath.Join(providersDir, "anthropic", "models")
	if err := os.MkdirAll(anthropicDir, 0o755); err != nil {
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
	if err := os.WriteFile(filepath.Join(anthropicDir, "claude-sonnet-4-5.yaml"), anthropicYAML, 0o644); err != nil {
		t.Fatalf("write anthropic yaml: %v", err)
	}

	// Run build
	if err := Build(providersDir, distDir); err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Verify output
	outputPath := filepath.Join(distDir, "catalog.json")
	data, err := os.ReadFile(outputPath)
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

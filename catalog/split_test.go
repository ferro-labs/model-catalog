package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"gpt-4o", "gpt-4o"},
		{"claude-sonnet-4-5", "claude-sonnet-4-5"},
		{"anthropic.claude-sonnet-4-5-v1:0", "anthropic.claude-sonnet-4-5-v1_0"},
		{"meta-llama/Meta-Llama-3.1-70B", "meta-llama__Meta-Llama-3.1-70B"},
		{"1024-x-1024/50-steps/bedrock/amazon.nova-canvas-v1:0", "1024-x-1024__50-steps__bedrock__amazon.nova-canvas-v1_0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSplitCatalog(t *testing.T) {
	catalogJSON := `{
  "openai/gpt-4o": {
    "provider": "openai",
    "model_id": "gpt-4o",
    "display_name": "GPT-4o",
    "mode": "chat",
    "context_window": 128000,
    "max_output_tokens": 4096,
    "pricing": {},
    "capabilities": {},
    "lifecycle": {"status": "active"},
    "source": "openai",
    "updated_at": "2025-01-01",
    "tier": "flagship"
  },
  "anthropic/claude-sonnet-4-5": {
    "provider": "anthropic",
    "model_id": "claude-sonnet-4-5",
    "display_name": "Claude Sonnet 4.5",
    "mode": "chat",
    "context_window": 200000,
    "max_output_tokens": 8192,
    "pricing": {},
    "capabilities": {},
    "lifecycle": {"status": "active"},
    "source": "anthropic",
    "updated_at": "2025-01-01",
    "tier": "flagship"
  }
}`

	outputDir := t.TempDir()

	if err := Split([]byte(catalogJSON), outputDir); err != nil {
		t.Fatalf("Split() error: %v", err)
	}

	// Verify openai model file exists
	openaiPath := filepath.Join(outputDir, "openai", "models", "gpt-4o.yaml")
	data, err := os.ReadFile(filepath.Clean(openaiPath))
	if err != nil {
		t.Fatalf("expected file %s to exist: %v", openaiPath, err)
	}
	entry, err := ReadModelYAML(data)
	if err != nil {
		t.Fatalf("failed to parse openai YAML: %v", err)
	}
	if entry.Provider != "openai" {
		t.Errorf("openai entry.Provider = %q, want %q", entry.Provider, "openai")
	}
	if entry.ModelID != "gpt-4o" {
		t.Errorf("openai entry.ModelID = %q, want %q", entry.ModelID, "gpt-4o")
	}

	// Verify anthropic model file exists
	anthropicPath := filepath.Join(outputDir, "anthropic", "models", "claude-sonnet-4-5.yaml")
	data, err = os.ReadFile(filepath.Clean(anthropicPath))
	if err != nil {
		t.Fatalf("expected file %s to exist: %v", anthropicPath, err)
	}
	entry, err = ReadModelYAML(data)
	if err != nil {
		t.Fatalf("failed to parse anthropic YAML: %v", err)
	}
	if entry.Provider != "anthropic" {
		t.Errorf("anthropic entry.Provider = %q, want %q", entry.Provider, "anthropic")
	}
	if entry.ModelID != "claude-sonnet-4-5" {
		t.Errorf("anthropic entry.ModelID = %q, want %q", entry.ModelID, "claude-sonnet-4-5")
	}
}

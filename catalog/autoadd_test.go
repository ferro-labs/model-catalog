package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutoAdd(t *testing.T) {
	t.Run("adds model when provider folder exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		provDir := filepath.Join(tmpDir, "openai", "models")
		os.MkdirAll(provDir, 0o755)

		input := 5.0
		output := 15.0
		candidates := []AutoAddCandidate{{
			Provider:   "openai",
			ModelID:    "gpt-new",
			InputPerM:  &input,
			OutputPerM: &output,
			Sources:    []string{"models_dev", "openrouter"},
		}}

		result, err := AutoAdd(tmpDir, candidates, false)
		if err != nil {
			t.Fatalf("AutoAdd: %v", err)
		}

		if result.Added != 1 {
			t.Errorf("added: got %d, want 1", result.Added)
		}

		data, err := os.ReadFile(filepath.Join(provDir, "gpt-new.yaml"))
		if err != nil {
			t.Fatalf("read generated file: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, "model_id: gpt-new") {
			t.Error("missing model_id")
		}
		if !strings.Contains(content, "source: auto:models_dev+openrouter") {
			t.Error("missing source tag")
		}
		if !strings.Contains(content, "status: preview") {
			t.Error("missing preview status")
		}
		if !strings.Contains(content, "input_per_m_tokens: 5") {
			t.Error("missing input pricing")
		}
	})

	t.Run("skips when provider folder missing", func(t *testing.T) {
		tmpDir := t.TempDir()

		candidates := []AutoAddCandidate{{
			Provider: "nonexistent",
			ModelID:  "model-1",
			Sources:  []string{"openrouter"},
		}}

		result, err := AutoAdd(tmpDir, candidates, false)
		if err != nil {
			t.Fatalf("AutoAdd: %v", err)
		}

		if result.NoProvider != 1 {
			t.Errorf("noProvider: got %d, want 1", result.NoProvider)
		}
		if result.Added != 0 {
			t.Errorf("added: got %d, want 0", result.Added)
		}
	})

	t.Run("skips when file already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		provDir := filepath.Join(tmpDir, "openai", "models")
		os.MkdirAll(provDir, 0o755)
		os.WriteFile(filepath.Join(provDir, "gpt-exists.yaml"), []byte("existing"), 0o644)

		candidates := []AutoAddCandidate{{
			Provider: "openai",
			ModelID:  "gpt-exists",
			Sources:  []string{"openrouter"},
		}}

		result, err := AutoAdd(tmpDir, candidates, false)
		if err != nil {
			t.Fatalf("AutoAdd: %v", err)
		}

		if result.Skipped != 1 {
			t.Errorf("skipped: got %d, want 1", result.Skipped)
		}
	})

	t.Run("dry run does not write files", func(t *testing.T) {
		tmpDir := t.TempDir()
		provDir := filepath.Join(tmpDir, "openai", "models")
		os.MkdirAll(provDir, 0o755)

		input := 1.0
		candidates := []AutoAddCandidate{{
			Provider:  "openai",
			ModelID:   "gpt-dry",
			InputPerM: &input,
			Sources:   []string{"openrouter"},
		}}

		result, err := AutoAdd(tmpDir, candidates, true)
		if err != nil {
			t.Fatalf("AutoAdd: %v", err)
		}

		if result.Added != 1 {
			t.Errorf("added: got %d, want 1", result.Added)
		}

		_, err = os.Stat(filepath.Join(provDir, "gpt-dry.yaml"))
		if err == nil {
			t.Error("file should not exist in dry-run mode")
		}
	})
}

package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestModel(t *testing.T, dir string, provider string, filename string, entry Entry) {
	t.Helper()
	modelsDir := filepath.Join(dir, provider, "models")
	if err := os.MkdirAll(modelsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", modelsDir, err)
	}
	data, err := WriteModelYAML(entry)
	if err != nil {
		t.Fatalf("marshal entry: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modelsDir, filename), data, 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
}

func TestValidateProviders_ValidEntries(t *testing.T) {
	tmpDir := t.TempDir()

	writeTestModel(t, tmpDir, "openai", "gpt-4o.yaml", Entry{
		Provider:    "openai",
		ModelID:     "gpt-4o",
		DisplayName: "GPT-4o",
		Mode:        "chat",
		Lifecycle:   Lifecycle{Status: "ga"},
		Tier:        "flagship",
	})

	writeTestModel(t, tmpDir, "anthropic", "claude-sonnet.yaml", Entry{
		Provider:    "anthropic",
		ModelID:     "claude-sonnet-4-5",
		DisplayName: "Claude Sonnet 4.5",
		Mode:        "chat",
		Lifecycle:   Lifecycle{Status: "ga"},
		Tier:        "standard",
	})

	writeTestModel(t, tmpDir, "openai", "text-embedding-3-small.yaml", Entry{
		Provider:    "openai",
		ModelID:     "text-embedding-3-small",
		DisplayName: "Text Embedding 3 Small",
		Mode:        "embedding",
		Lifecycle:   Lifecycle{Status: "ga"},
		Tier:        "standard",
	})

	errs, err := Validate(tmpDir)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected error: %s: %s: %s", e.File, e.Field, e.Message)
		}
	}
}

func TestValidateProviders_InvalidMode(t *testing.T) {
	tmpDir := t.TempDir()

	writeTestModel(t, tmpDir, "openai", "bad-mode.yaml", Entry{
		Provider:    "openai",
		ModelID:     "bad-mode-model",
		DisplayName: "Bad Mode Model",
		Mode:        "video",
		Lifecycle:   Lifecycle{Status: "ga"},
		Tier:        "standard",
	})

	errs, err := Validate(tmpDir)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	if errs[0].Field != "mode" {
		t.Errorf("expected field 'mode', got %q", errs[0].Field)
	}
	if !strings.Contains(errs[0].Message, "video") {
		t.Errorf("expected message to contain 'video', got %q", errs[0].Message)
	}
}

func TestValidateProviders_EmptyProvider(t *testing.T) {
	tmpDir := t.TempDir()

	writeTestModel(t, tmpDir, "openai", "no-provider.yaml", Entry{
		Provider:    "",
		ModelID:     "some-model",
		DisplayName: "Some Model",
		Mode:        "chat",
		Lifecycle:   Lifecycle{Status: "ga"},
		Tier:        "standard",
	})

	errs, err := Validate(tmpDir)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	if errs[0].Field != "provider" {
		t.Errorf("expected field 'provider', got %q", errs[0].Field)
	}
	if !strings.Contains(errs[0].Message, "required") {
		t.Errorf("expected message to contain 'required', got %q", errs[0].Message)
	}
}

func TestValidateProviders_WrongProviderFolder(t *testing.T) {
	tmpDir := t.TempDir()

	writeTestModel(t, tmpDir, "openai", "misplaced.yaml", Entry{
		Provider:    "anthropic",
		ModelID:     "claude-sonnet",
		DisplayName: "Claude Sonnet",
		Mode:        "chat",
		Lifecycle:   Lifecycle{Status: "ga"},
		Tier:        "standard",
	})

	errs, err := Validate(tmpDir)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	if errs[0].Field != "provider" {
		t.Errorf("expected field 'provider', got %q", errs[0].Field)
	}
	if !strings.Contains(errs[0].Message, "does not match directory") {
		t.Errorf("expected message about directory mismatch, got %q", errs[0].Message)
	}
}

func TestValidateProviders_InvalidStatus(t *testing.T) {
	tmpDir := t.TempDir()

	writeTestModel(t, tmpDir, "openai", "bad-status.yaml", Entry{
		Provider:    "openai",
		ModelID:     "bad-status-model",
		DisplayName: "Bad Status Model",
		Mode:        "chat",
		Lifecycle:   Lifecycle{Status: "active"},
		Tier:        "standard",
	})

	errs, err := Validate(tmpDir)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	if errs[0].Field != "lifecycle.status" {
		t.Errorf("expected field 'lifecycle.status', got %q", errs[0].Field)
	}
	if !strings.Contains(errs[0].Message, "active") {
		t.Errorf("expected message to contain 'active', got %q", errs[0].Message)
	}
}

func TestValidateProviders_InvalidTier(t *testing.T) {
	tmpDir := t.TempDir()

	writeTestModel(t, tmpDir, "openai", "bad-tier.yaml", Entry{
		Provider:    "openai",
		ModelID:     "bad-tier-model",
		DisplayName: "Bad Tier Model",
		Mode:        "chat",
		Lifecycle:   Lifecycle{Status: "ga"},
		Tier:        "premium",
	})

	errs, err := Validate(tmpDir)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	if errs[0].Field != "tier" {
		t.Errorf("expected field 'tier', got %q", errs[0].Field)
	}
	if !strings.Contains(errs[0].Message, "premium") {
		t.Errorf("expected message to contain 'premium', got %q", errs[0].Message)
	}
}

func TestValidateProviders_MultipleErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Entry with empty mode AND wrong provider directory
	writeTestModel(t, tmpDir, "openai", "multi-bad.yaml", Entry{
		Provider:    "wrong-provider",
		ModelID:     "some-model",
		DisplayName: "Some Model",
		Mode:        "",
		Lifecycle:   Lifecycle{Status: "ga"},
		Tier:        "standard",
	})

	errs, err := Validate(tmpDir)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(errs))
	}

	fields := make(map[string]bool)
	for _, e := range errs {
		fields[e.Field] = true
	}

	if !fields["mode"] {
		t.Error("expected error for field 'mode'")
	}
	if !fields["provider"] {
		t.Error("expected error for field 'provider'")
	}
}

func TestValidateProviders_AllModes(t *testing.T) {
	tmpDir := t.TempDir()

	modes := []string{"chat", "embedding", "image", "audio_in", "audio_out"}
	for _, mode := range modes {
		writeTestModel(t, tmpDir, "testprovider", mode+".yaml", Entry{
			Provider:    "testprovider",
			ModelID:     mode + "-model",
			DisplayName: mode + " Model",
			Mode:        mode,
			Lifecycle:   Lifecycle{Status: "ga"},
			Tier:        "standard",
		})
	}

	errs, err := Validate(tmpDir)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected error: %s: %s: %s", e.File, e.Field, e.Message)
		}
	}
}

func TestValidateProviders_AllStatuses(t *testing.T) {
	tmpDir := t.TempDir()

	statuses := []string{"preview", "ga", "deprecated", "sunset", "legacy"}
	for _, status := range statuses {
		writeTestModel(t, tmpDir, "testprovider", status+".yaml", Entry{
			Provider:    "testprovider",
			ModelID:     status + "-model",
			DisplayName: status + " Model",
			Mode:        "chat",
			Lifecycle:   Lifecycle{Status: status},
			Tier:        "standard",
		})
	}

	errs, err := Validate(tmpDir)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected error: %s: %s: %s", e.File, e.Field, e.Message)
		}
	}
}

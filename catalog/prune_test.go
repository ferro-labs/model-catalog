package catalog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPruneSunset(t *testing.T) {
	tmpDir := t.TempDir()
	modelsDir := filepath.Join(tmpDir, "openai", "models")
	if err := os.MkdirAll(modelsDir, 0o750); err != nil {
		t.Fatal(err)
	}

	oldDate := "2026-01-01"
	recentDate := "2026-04-01"
	writeEntry := func(name string, sunset *string) string {
		t.Helper()
		entry := Entry{
			Provider:        "openai",
			ModelID:         name,
			DisplayName:     name,
			Mode:            "chat",
			ContextWindow:   1,
			MaxOutputTokens: 1,
			Pricing:         Pricing{},
			Capabilities:    Capabilities{},
			Lifecycle:       Lifecycle{Status: "sunset", SunsetDate: sunset},
			Tier:            "standard",
		}
		data, err := WriteModelYAML(entry)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(modelsDir, name+".yaml")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}

	oldPath := writeEntry("old-model", &oldDate)
	recentPath := writeEntry("recent-model", &recentDate)
	keepPath := writeEntry("active-model", nil)

	cutoff := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	pruned, err := PruneSunset(tmpDir, cutoff, false)
	if err != nil {
		t.Fatalf("PruneSunset: %v", err)
	}
	if pruned != 1 {
		t.Fatalf("pruned got %d, want 1", pruned)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old model still exists or unexpected stat error: %v", err)
	}
	for _, path := range []string{recentPath, keepPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to remain: %v", path, err)
		}
	}
}

func TestPruneSunsetDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	modelsDir := filepath.Join(tmpDir, "openai", "models")
	if err := os.MkdirAll(modelsDir, 0o750); err != nil {
		t.Fatal(err)
	}

	sunset := "2026-01-01"
	entry := Entry{
		Provider:        "openai",
		ModelID:         "old-model",
		DisplayName:     "old-model",
		Mode:            "chat",
		ContextWindow:   1,
		MaxOutputTokens: 1,
		Pricing:         Pricing{},
		Capabilities:    Capabilities{},
		Lifecycle:       Lifecycle{Status: "sunset", SunsetDate: &sunset},
		Tier:            "standard",
	}
	data, err := WriteModelYAML(entry)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(modelsDir, "old-model.yaml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	cutoff := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	pruned, err := PruneSunset(tmpDir, cutoff, true)
	if err != nil {
		t.Fatalf("PruneSunset: %v", err)
	}
	if pruned != 1 {
		t.Fatalf("pruned got %d, want 1", pruned)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("dry run should leave file in place: %v", err)
	}
}

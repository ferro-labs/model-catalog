package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// setupTestProviders creates two providers with one model each in a temp directory
// and returns the providers dir and dist dir paths.
func setupTestProviders(t *testing.T) (providersDir, distDir string) {
	t.Helper()
	tmpDir := t.TempDir()
	providersDir = filepath.Join(tmpDir, "providers")
	distDir = filepath.Join(tmpDir, "dist")

	// openai provider
	openaiDir := filepath.Join(providersDir, "openai", "models")
	if err := os.MkdirAll(openaiDir, 0o750); err != nil {
		t.Fatalf("mkdir openai: %v", err)
	}
	openaiYAML, err := WriteModelYAML(Entry{
		Provider:      "openai",
		ModelID:       "gpt-4o",
		DisplayName:   "GPT-4o",
		Mode:          "chat",
		ContextWindow: 128000,
		Lifecycle:     Lifecycle{Status: "active"},
	})
	if err != nil {
		t.Fatalf("marshal openai: %v", err)
	}
	if err := os.WriteFile(filepath.Join(openaiDir, "gpt-4o.yaml"), openaiYAML, 0o600); err != nil {
		t.Fatalf("write openai yaml: %v", err)
	}

	// anthropic provider
	anthropicDir := filepath.Join(providersDir, "anthropic", "models")
	if err := os.MkdirAll(anthropicDir, 0o750); err != nil {
		t.Fatalf("mkdir anthropic: %v", err)
	}
	anthropicYAML, err := WriteModelYAML(Entry{
		Provider:      "anthropic",
		ModelID:       "claude-sonnet-4-5",
		DisplayName:   "Claude Sonnet 4.5",
		Mode:          "chat",
		ContextWindow: 200000,
		Lifecycle:     Lifecycle{Status: "active"},
	})
	if err != nil {
		t.Fatalf("marshal anthropic: %v", err)
	}
	if err := os.WriteFile(filepath.Join(anthropicDir, "claude-sonnet-4-5.yaml"), anthropicYAML, 0o600); err != nil {
		t.Fatalf("write anthropic yaml: %v", err)
	}

	return providersDir, distDir
}

func TestBuildGeneratesProviderSlices(t *testing.T) {
	providersDir, distDir := setupTestProviders(t)

	if err := Build(providersDir, distDir); err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Verify openai slice exists and contains correct entries
	openaiSlice := filepath.Join(distDir, "providers", "openai.json")
	openaiData, err := os.ReadFile(filepath.Clean(openaiSlice))
	if err != nil {
		t.Fatalf("read openai slice: %v", err)
	}

	var openaiEntries map[string]Entry
	if err := json.Unmarshal(openaiData, &openaiEntries); err != nil {
		t.Fatalf("parse openai slice: %v", err)
	}

	if len(openaiEntries) != 1 {
		t.Errorf("openai slice: expected 1 entry, got %d", len(openaiEntries))
	}
	if _, ok := openaiEntries["openai/gpt-4o"]; !ok {
		t.Error("openai slice missing key openai/gpt-4o")
	}

	// Verify anthropic slice exists and contains correct entries
	anthropicSlice := filepath.Join(distDir, "providers", "anthropic.json")
	anthropicData, err := os.ReadFile(filepath.Clean(anthropicSlice))
	if err != nil {
		t.Fatalf("read anthropic slice: %v", err)
	}

	var anthropicEntries map[string]Entry
	if err := json.Unmarshal(anthropicData, &anthropicEntries); err != nil {
		t.Fatalf("parse anthropic slice: %v", err)
	}

	if len(anthropicEntries) != 1 {
		t.Errorf("anthropic slice: expected 1 entry, got %d", len(anthropicEntries))
	}
	if _, ok := anthropicEntries["anthropic/claude-sonnet-4-5"]; !ok {
		t.Error("anthropic slice missing key anthropic/claude-sonnet-4-5")
	}

	// Verify no cross-contamination
	if _, ok := openaiEntries["anthropic/claude-sonnet-4-5"]; ok {
		t.Error("openai slice should not contain anthropic entries")
	}
	if _, ok := anthropicEntries["openai/gpt-4o"]; ok {
		t.Error("anthropic slice should not contain openai entries")
	}
}

func TestBuildGeneratesManifest(t *testing.T) {
	providersDir, distDir := setupTestProviders(t)

	if err := Build(providersDir, distDir); err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	manifestPath := filepath.Join(distDir, "manifest.json")
	manifestData, err := os.ReadFile(filepath.Clean(manifestPath))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	// Check schema version
	if manifest.SchemaVersion != 1 {
		t.Errorf("schema_version = %d, want 1", manifest.SchemaVersion)
	}

	// Check version format (CalVer)
	if len(manifest.Version) < 8 || manifest.Version[0] != 'v' {
		t.Errorf("version %q does not look like CalVer", manifest.Version)
	}

	// Check generated_at is non-empty
	if manifest.GeneratedAt == "" {
		t.Error("generated_at is empty")
	}

	// Check catalog_sha256 is non-empty hex
	if len(manifest.CatalogSHA256) != 64 {
		t.Errorf("catalog_sha256 length = %d, want 64", len(manifest.CatalogSHA256))
	}

	if want := "/v1/" + manifest.CatalogSHA256 + ".json"; manifest.CatalogURL != want {
		t.Errorf("catalog_url = %q, want %q", manifest.CatalogURL, want)
	}
	if _, err := os.Stat(filepath.Join(distDir, manifest.CatalogSHA256+".json")); err != nil {
		t.Errorf("content-addressed catalog: %v", err)
	}

	// Check stats
	if manifest.Stats.TotalModels != 2 {
		t.Errorf("total_models = %d, want 2", manifest.Stats.TotalModels)
	}
	if manifest.Stats.TotalProviders != 2 {
		t.Errorf("total_providers = %d, want 2", manifest.Stats.TotalProviders)
	}

	// Check providers array
	if len(manifest.Providers) != 2 {
		t.Fatalf("providers count = %d, want 2", len(manifest.Providers))
	}

	// Providers should be sorted alphabetically
	if manifest.Providers[0].ID != "anthropic" { //nolint:goconst // test data
		t.Errorf("first provider = %q, want anthropic", manifest.Providers[0].ID)
	}
	if manifest.Providers[1].ID != "openai" {
		t.Errorf("second provider = %q, want openai", manifest.Providers[1].ID)
	}

	// Each provider should have model_count = 1
	for _, p := range manifest.Providers {
		if p.ModelCount != 1 {
			t.Errorf("provider %s model_count = %d, want 1", p.ID, p.ModelCount)
		}
		if len(p.SHA256) != 64 {
			t.Errorf("provider %s sha256 length = %d, want 64", p.ID, len(p.SHA256))
		}
		if want := "/v1/providers/" + p.ID + "/" + p.SHA256 + ".json"; p.URL != want {
			t.Errorf("provider %s url = %q, want %q", p.ID, p.URL, want)
		}
		if _, err := os.Stat(filepath.Join(distDir, "providers", p.ID, p.SHA256+".json")); err != nil {
			t.Errorf("content-addressed provider %s: %v", p.ID, err)
		}
	}
}

func TestBuildIncludesGitSHA(t *testing.T) {
	providersDir, distDir := setupTestProviders(t)

	if err := buildWithVersionAndGitSHA(providersDir, distDir, "v2026.07.13", "abc123"); err != nil {
		t.Fatalf("buildWithVersionAndGitSHA() error: %v", err)
	}

	manifest, err := ReadManifest(distDir)
	if err != nil {
		t.Fatalf("ReadManifest() error: %v", err)
	}
	if manifest.GitSHA != "abc123" {
		t.Errorf("git_sha = %q, want abc123", manifest.GitSHA)
	}
}

func TestManifestSHA256Matches(t *testing.T) {
	providersDir, distDir := setupTestProviders(t)

	if err := Build(providersDir, distDir); err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	// Read catalog.json and compute SHA-256 manually
	catalogData, err := os.ReadFile(filepath.Clean(filepath.Join(distDir, "catalog.json")))
	if err != nil {
		t.Fatalf("read catalog.json: %v", err)
	}
	h := sha256.Sum256(catalogData)
	expectedHash := hex.EncodeToString(h[:])

	// Read manifest and compare
	manifestData, err := os.ReadFile(filepath.Clean(filepath.Join(distDir, "manifest.json")))
	if err != nil {
		t.Fatalf("read manifest.json: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	if manifest.CatalogSHA256 != expectedHash {
		t.Errorf("catalog_sha256 mismatch:\n  manifest: %s\n  computed: %s", manifest.CatalogSHA256, expectedHash)
	}

	// Also verify provider slice SHA-256 values
	for _, p := range manifest.Providers {
		sliceData, err := os.ReadFile(filepath.Clean(filepath.Join(distDir, "providers", p.ID+".json")))
		if err != nil {
			t.Fatalf("read provider slice %s: %v", p.ID, err)
		}
		sliceHash := sha256.Sum256(sliceData)
		sliceHashHex := hex.EncodeToString(sliceHash[:])
		if p.SHA256 != sliceHashHex {
			t.Errorf("provider %s sha256 mismatch:\n  manifest: %s\n  computed: %s", p.ID, p.SHA256, sliceHashHex)
		}
	}
}

package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

// writeProvModel writes a model YAML with a literal filename (unlike the
// writeModel helper in applyprices_test.go, which derives the filename from
// a sanitized model ID); named distinctly to avoid colliding with it.
func writeProvModel(t *testing.T, dir, provider, file, body string) {
	t.Helper()
	md := filepath.Join(dir, provider, "models")
	if err := os.MkdirAll(md, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(md, file), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateProvenance(t *testing.T) {
	dir := t.TempDir()
	// eligible: has source + date, no sources block
	writeProvModel(t, dir, "openai", "a.yaml",
		"provider: openai\nmodel_id: a\ndisplay_name: A\nmode: chat\nlifecycle:\n  status: ga\nsource: \"https://x\"\nupdated_at: \"2026-08-04\"\ntier: standard\n")
	// wrapper: has extends, no source → skipped
	writeProvModel(t, dir, "bedrock", "b.yaml",
		"extends: openai/a\nprovider: bedrock\nmodel_id: a\ntier: standard\n")
	// no date → skipped
	writeProvModel(t, dir, "openai", "c.yaml",
		"provider: openai\nmodel_id: c\ndisplay_name: C\nmode: chat\nlifecycle:\n  status: ga\nsource: \"https://x\"\nupdated_at: \"\"\ntier: standard\n")

	migrated, skipped, err := MigrateProvenance(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if migrated != 1 {
		t.Errorf("migrated = %d, want 1", migrated)
	}
	if skipped != 1 { // c.yaml; the wrapper is silently skipped (no source)
		t.Errorf("skipped = %d, want 1", skipped)
	}
	got, err := os.ReadFile(filepath.Join(dir, "openai", "models", "a.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	e, err := ReadModelYAML(got)
	if err != nil {
		t.Fatal(err)
	}
	if e.Sources == nil || e.Sources.Pricing == nil {
		t.Fatalf("a.yaml not migrated: %s", got)
	}
	if e.Sources.Pricing.VerifiedBy != "import:v1" || e.Sources.Pricing.Confidence != "low" {
		t.Errorf("wrong baseline provenance: %+v", *e.Sources.Pricing)
	}
	// url/verified_at must come from the right fixture field, not swapped
	if e.Sources.Pricing.URL != "https://x" {
		t.Errorf("Pricing.URL = %q, want %q", e.Sources.Pricing.URL, "https://x")
	}
	if e.Sources.Pricing.VerifiedAt != "2026-08-04" {
		t.Errorf("Pricing.VerifiedAt = %q, want %q", e.Sources.Pricing.VerifiedAt, "2026-08-04")
	}
	// idempotent: second run migrates nothing
	migrated2, _, _ := MigrateProvenance(dir, false)
	if migrated2 != 0 {
		t.Errorf("second run migrated %d, want 0 (not idempotent)", migrated2)
	}

	t.Run("dry-run does not write", func(t *testing.T) {
		dryDir := t.TempDir()
		writeProvModel(t, dryDir, "openai", "a.yaml",
			"provider: openai\nmodel_id: a\ndisplay_name: A\nmode: chat\nlifecycle:\n  status: ga\nsource: \"https://x\"\nupdated_at: \"2026-08-04\"\ntier: standard\n")

		dryMigrated, _, err := MigrateProvenance(dryDir, true)
		if err != nil {
			t.Fatal(err)
		}
		if dryMigrated == 0 {
			t.Errorf("dry-run migrated = %d, want > 0", dryMigrated)
		}
		got, err := os.ReadFile(filepath.Join(dryDir, "openai", "models", "a.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		e, err := ReadModelYAML(got)
		if err != nil {
			t.Fatal(err)
		}
		if e.Sources != nil {
			t.Errorf("dry-run wrote to disk: Sources = %+v, want nil", e.Sources)
		}
	})
}

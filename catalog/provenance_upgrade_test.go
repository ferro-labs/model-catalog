package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// baselineModel returns a model YAML with a trailing migrate-style sources block
// (verified_by import:v1, confidence low) — the shape ApplyProvenanceUpgrades
// upgrades in place.
func baselineModel(confidence string) string {
	return "provider: openai\nmodel_id: gpt-x\ndisplay_name: GPT-X\nmode: chat\n" +
		"lifecycle:\n    status: ga\nsource: \"https://x\"\nupdated_at: \"2026-01-01\"\ntier: standard\n" +
		"sources:\n    pricing:\n        url: \"https://x\"\n        verified_at: \"2026-01-01\"\n" +
		"        verified_by: import:v1\n        confidence: " + confidence + "\n"
}

func upgrade(confidence string) ProvenanceUpgrade {
	return ProvenanceUpgrade{
		Provider: "openai", ModelID: "gpt-x", Group: "pricing",
		Prov: Provenance{
			URL: "https://openrouter.ai/api/v1/models", VerifiedAt: "2026-08-07",
			Confidence: confidence, VerifiedBy: "scraper-openrouter",
			SnapshotSHA256: strptr(strings.Repeat("a", 64)), SnapshotBranch: strptr("snapshots"),
		},
	}
}

func TestApplyProvenanceUpgrades(t *testing.T) {
	dir := t.TempDir()
	writeProvModel(t, dir, "openai", "gpt-x.yaml", baselineModel("low"))

	applied, skipped, err := ApplyProvenanceUpgrades(dir, []ProvenanceUpgrade{upgrade("medium")})
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 || skipped != 0 {
		t.Fatalf("applied=%d skipped=%d, want 1/0", applied, skipped)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "openai", "models", "gpt-x.yaml"))
	e, err := ReadModelYAML(got)
	if err != nil {
		t.Fatalf("re-parse failed: %v\n%s", err, got)
	}
	p := e.Sources.Pricing
	if p.VerifiedBy != "scraper-openrouter" || p.Confidence != "medium" {
		t.Errorf("provenance not upgraded: %+v", *p)
	}
	if p.SnapshotSHA256 == nil || *p.SnapshotSHA256 != strings.Repeat("a", 64) {
		t.Errorf("snapshot sha not recorded: %+v", *p)
	}
	if p.SnapshotBranch == nil || *p.SnapshotBranch != "snapshots" {
		t.Errorf("snapshot branch not recorded: %+v", *p)
	}
	// Non-provenance bytes must survive the splice untouched.
	if !strings.Contains(string(got), "model_id: gpt-x") || !strings.Contains(string(got), "tier: standard") {
		t.Errorf("splice clobbered unrelated fields:\n%s", got)
	}
}

func TestApplyProvenanceUpgradesIdempotent(t *testing.T) {
	dir := t.TempDir()
	writeProvModel(t, dir, "openai", "gpt-x.yaml", baselineModel("low"))

	// First run upgrades the import:v1/low baseline.
	if applied, _, err := ApplyProvenanceUpgrades(dir, []ProvenanceUpgrade{upgrade("medium")}); err != nil || applied != 1 {
		t.Fatalf("first run applied=%d err=%v, want 1/nil", applied, err)
	}
	// Re-running the identical upgrade (same confidence, url, verified_by, snapshot)
	// is a no-op — only verified_at would change, so the weekly PR stays quiet.
	applied, skipped, err := ApplyProvenanceUpgrades(dir, []ProvenanceUpgrade{upgrade("medium")})
	if err != nil {
		t.Fatal(err)
	}
	if applied != 0 || skipped != 1 {
		t.Fatalf("re-run applied=%d skipped=%d, want 0/1 (idempotent)", applied, skipped)
	}

	// A changed snapshot (content moved) IS re-applied.
	up := upgrade("medium")
	up.Prov.SnapshotSHA256 = strptr(strings.Repeat("b", 64))
	if applied, _, err := ApplyProvenanceUpgrades(dir, []ProvenanceUpgrade{up}); err != nil || applied != 1 {
		t.Fatalf("changed-snapshot run applied=%d err=%v, want 1/nil", applied, err)
	}
}

func TestApplyProvenanceUpgradesContextGroup(t *testing.T) {
	dir := t.TempDir()
	writeProvModel(t, dir, "openai", "gpt-x.yaml", baselineModel("low"))

	// A model can receive both a pricing and a context upgrade in one call. Each
	// upgrade re-reads the file, so the second group is merged onto the first.
	ctxUp := ProvenanceUpgrade{
		Provider: "openai", ModelID: "gpt-x", Group: "context",
		Prov: Provenance{
			URL: "https://openrouter.ai/api/v1/models", VerifiedAt: "2026-08-07",
			Confidence: "high", VerifiedBy: "scraper-openrouter",
			SnapshotSHA256: strptr(strings.Repeat("c", 64)), SnapshotBranch: strptr("snapshots"),
		},
	}
	applied, skipped, err := ApplyProvenanceUpgrades(dir, []ProvenanceUpgrade{upgrade("medium"), ctxUp})
	if err != nil {
		t.Fatal(err)
	}
	if applied != 2 || skipped != 0 {
		t.Fatalf("applied=%d skipped=%d, want 2/0", applied, skipped)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "openai", "models", "gpt-x.yaml"))
	e, err := ReadModelYAML(got)
	if err != nil {
		t.Fatalf("re-parse failed: %v\n%s", err, got)
	}
	if e.Sources.Pricing == nil || e.Sources.Pricing.Confidence != "medium" {
		t.Errorf("pricing group lost/wrong: %+v", e.Sources)
	}
	if e.Sources.Context == nil || e.Sources.Context.Confidence != "high" {
		t.Errorf("context group not written: %+v", e.Sources)
	}
	if e.Sources.Context.SnapshotSHA256 == nil || *e.Sources.Context.SnapshotSHA256 != strings.Repeat("c", 64) {
		t.Errorf("context snapshot sha not recorded: %+v", e.Sources.Context)
	}
}

func TestApplyProvenanceUpgradesNeverDowngrades(t *testing.T) {
	dir := t.TempDir()
	writeProvModel(t, dir, "openai", "gpt-x.yaml", baselineModel("high"))

	applied, skipped, err := ApplyProvenanceUpgrades(dir, []ProvenanceUpgrade{upgrade("low")})
	if err != nil {
		t.Fatal(err)
	}
	if applied != 0 || skipped != 1 {
		t.Fatalf("applied=%d skipped=%d, want 0/1 (monotonicity)", applied, skipped)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "openai", "models", "gpt-x.yaml"))
	e, _ := ReadModelYAML(got)
	if e.Sources.Pricing.VerifiedBy != "import:v1" {
		t.Errorf("high-confidence group was downgraded: %+v", *e.Sources.Pricing)
	}
}

func TestApplyProvenanceUpgradesSkipsNonTrailingSources(t *testing.T) {
	dir := t.TempDir()
	// sources: is NOT the final top-level block — a trailing splice would eat `tier:`.
	body := "provider: openai\nmodel_id: gpt-x\ndisplay_name: GPT-X\nmode: chat\n" +
		"sources:\n    pricing:\n        url: \"https://x\"\n        verified_at: \"2026-01-01\"\n" +
		"        verified_by: import:v1\n        confidence: low\n" +
		"tier: standard\n"
	writeProvModel(t, dir, "openai", "gpt-x.yaml", body)

	applied, skipped, err := ApplyProvenanceUpgrades(dir, []ProvenanceUpgrade{upgrade("high")})
	if err != nil {
		t.Fatal(err)
	}
	if applied != 0 || skipped != 1 {
		t.Fatalf("applied=%d skipped=%d, want 0/1 (unsafe splice skipped)", applied, skipped)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "openai", "models", "gpt-x.yaml"))
	if !strings.Contains(string(got), "tier: standard") {
		t.Errorf("skipped file was modified and lost tier:\n%s", got)
	}
}

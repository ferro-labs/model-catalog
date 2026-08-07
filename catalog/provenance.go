package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MigrateProvenance seeds a baseline sources.pricing block on every non-wrapper
// model that has a legacy flat source + updated_at but no sources block yet.
// Imported provenance is labeled import:v1 / confidence low so scrapers can
// upgrade it later. Wrappers are skipped: they inherit provenance from their
// base via extends resolution. Models with no source or a non-YYYY-MM-DD date
// are counted as skipped and left untouched.
//
// This deliberately does NOT round-trip through WriteModelYAML: the
// providers/ tree has drifted from yaml.Marshal's canonical form (quoted
// scalars that don't need quoting, ".0" on whole-number pricing), so a full
// re-marshal would rewrite thousands of unrelated bytes. Instead it appends
// the new sources: block as text, byte-for-byte preserving every existing
// line — mirroring the %q text-splice already used by BackfillSource in
// catalog/backfill.go.
func MigrateProvenance(providersDir string, dryRun bool) (migrated, skipped int, err error) {
	matches, err := filepath.Glob(filepath.Join(providersDir, "*", "models", "*.yaml"))
	if err != nil {
		return 0, 0, fmt.Errorf("glob: %w", err)
	}
	for _, path := range matches {
		data, readErr := os.ReadFile(filepath.Clean(path))
		if readErr != nil {
			return migrated, skipped, fmt.Errorf("read %s: %w", path, readErr)
		}
		entry, parseErr := ReadModelYAML(data)
		if parseErr != nil {
			continue // validate/lint own malformed files
		}
		if entry.Extends != "" || entry.Sources != nil {
			continue // wrapper (inherits) or already migrated
		}
		if entry.Source == "" || !dateRe.MatchString(entry.UpdatedAt) {
			skipped++
			continue
		}
		if dryRun {
			migrated++
			continue
		}
		content := string(data)
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		block := fmt.Sprintf("sources:\n    pricing:\n        url: %q\n        verified_at: %q\n        verified_by: import:v1\n        confidence: low\n", entry.Source, entry.UpdatedAt)
		if writeErr := os.WriteFile(filepath.Clean(path), []byte(content+block), 0o600); writeErr != nil { //nolint:gosec // path from filepath.Glob
			return migrated, skipped, fmt.Errorf("write %s: %w", path, writeErr)
		}
		migrated++
	}
	return migrated, skipped, nil
}

// ProvenanceUpgrade upgrades one field-group's provenance for one model.
type ProvenanceUpgrade struct {
	Provider string
	ModelID  string
	Group    string // "pricing" | "capabilities" | "context" | "lifecycle"
	Prov     Provenance
}

var confidenceRank = map[string]int{"low": 0, "medium": 1, "high": 2}

// ApplyProvenanceUpgrades rewrites the trailing sources: block of each named
// model's YAML to record upgraded provenance. It preserves every other byte of
// the file (mirroring MigrateProvenance's text-splice discipline, since the
// providers/ tree has drifted from yaml.Marshal's canonical form) and never
// downgrades a group: an upgrade applies only when its confidence is >= the
// existing group's. Files whose sources: block is not the final top-level block
// are skipped with a warning rather than corrupted.
func ApplyProvenanceUpgrades(providersDir string, ups []ProvenanceUpgrade) (applied, skipped int, err error) {
	for _, up := range ups {
		path := filepath.Join(providersDir, up.Provider, "models", SanitizeFilename(up.ModelID)+".yaml")
		data, readErr := os.ReadFile(filepath.Clean(path))
		if readErr != nil {
			return applied, skipped, fmt.Errorf("read %s: %w", path, readErr)
		}
		entry, parseErr := ReadModelYAML(data)
		if parseErr != nil {
			skipped++
			continue
		}
		cur := entry.Sources
		if cur == nil {
			cur = &Sources{}
		}
		if existing := groupProv(cur, up.Group); existing != nil {
			if confidenceRank[up.Prov.Confidence] < confidenceRank[existing.Confidence] {
				skipped++ // monotonicity: never downgrade a verified group
				continue
			}
			if sameProvenanceExceptDate(existing, &up.Prov) {
				skipped++ // idempotent: only verified_at would change — skip the churn
				continue
			}
		}
		merged := *cur
		prov := up.Prov
		setGroupProv(&merged, up.Group, &prov)

		newContent, ok := spliceTrailingSources(string(data), &merged)
		if !ok {
			fmt.Fprintf(os.Stderr, "  skip %s: sources block is not the final top-level block\n", path)
			skipped++
			continue
		}
		if writeErr := os.WriteFile(filepath.Clean(path), []byte(newContent), 0o600); writeErr != nil { //nolint:gosec // path from provider/model_id
			return applied, skipped, fmt.Errorf("write %s: %w", path, writeErr)
		}
		applied++
	}
	return applied, skipped, nil
}

// sameProvenanceExceptDate reports whether two Provenance values are identical
// apart from VerifiedAt. Used to make re-verification idempotent: if the weekly
// run would only bump the date, it is skipped so the PR stays quiet.
func sameProvenanceExceptDate(a, b *Provenance) bool {
	return a.URL == b.URL &&
		a.Confidence == b.Confidence &&
		a.VerifiedBy == b.VerifiedBy &&
		ptrStrEq(a.SnapshotSHA256, b.SnapshotSHA256) &&
		ptrStrEq(a.SnapshotBranch, b.SnapshotBranch)
}

func ptrStrEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func groupProv(s *Sources, group string) *Provenance {
	switch group {
	case "pricing":
		return s.Pricing
	case "capabilities":
		return s.Capabilities
	case "context":
		return s.Context
	case "lifecycle":
		return s.Lifecycle
	}
	return nil
}

func setGroupProv(s *Sources, group string, p *Provenance) {
	switch group {
	case "pricing":
		s.Pricing = p
	case "capabilities":
		s.Capabilities = p
	case "context":
		s.Context = p
	case "lifecycle":
		s.Lifecycle = p
	}
}

// spliceTrailingSources replaces a trailing top-level sources: block with a
// freshly rendered one, or appends it when absent. It returns ok=false when a
// sources: line exists but is followed by another top-level key (a column-0,
// non-blank line), which would make a splice-to-EOF unsafe.
func spliceTrailingSources(content string, s *Sources) (string, bool) {
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	rendered := renderSources(s)

	lines := strings.Split(content, "\n")
	idx := -1
	for i, ln := range lines {
		if ln == "sources:" {
			idx = i
			break
		}
	}
	if idx == -1 {
		return content + rendered, true
	}
	for _, ln := range lines[idx+1:] {
		if ln == "" {
			continue
		}
		if ln[0] != ' ' && ln[0] != '\t' {
			return "", false // another top-level key follows sources:
		}
	}
	return strings.Join(lines[:idx], "\n") + "\n" + rendered, true
}

// renderSources emits a sources: block matching MigrateProvenance's style
// (4-space indent, quoted url/verified_at). Only populated groups are written.
func renderSources(s *Sources) string {
	var b strings.Builder
	b.WriteString("sources:\n")
	writeGroup := func(name string, p *Provenance) {
		if p == nil {
			return
		}
		fmt.Fprintf(&b, "    %s:\n", name)
		fmt.Fprintf(&b, "        url: %q\n", p.URL)
		fmt.Fprintf(&b, "        verified_at: %q\n", p.VerifiedAt)
		if p.VerifiedBy != "" {
			fmt.Fprintf(&b, "        verified_by: %s\n", p.VerifiedBy)
		}
		if p.SnapshotSHA256 != nil {
			fmt.Fprintf(&b, "        snapshot_sha256: %s\n", *p.SnapshotSHA256)
		}
		if p.SnapshotBranch != nil {
			fmt.Fprintf(&b, "        snapshot_branch: %s\n", *p.SnapshotBranch)
		}
		fmt.Fprintf(&b, "        confidence: %s\n", p.Confidence)
	}
	writeGroup("pricing", s.Pricing)
	writeGroup("capabilities", s.Capabilities)
	writeGroup("context", s.Context)
	writeGroup("lifecycle", s.Lifecycle)
	return b.String()
}

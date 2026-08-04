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

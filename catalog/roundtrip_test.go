package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTripRealCatalog(t *testing.T) {
	// Use CATALOG_TEST_SOURCE env var if set, otherwise fall back to dist/catalog.json
	// relative to repo root. Skip if neither exists.
	catalogPath := os.Getenv("CATALOG_TEST_SOURCE")
	if catalogPath == "" {
		candidates := []string{
			"dist/catalog.json",
			"../dist/catalog.json",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				catalogPath = c
				break
			}
		}
	}
	if catalogPath == "" {
		t.Skip("no catalog found: set CATALOG_TEST_SOURCE or run from repo root with dist/catalog.json")
	}

	originalData, err := os.ReadFile(filepath.Clean(catalogPath))
	if err != nil {
		t.Skipf("cannot read catalog at %s: %v", catalogPath, err)
	}

	// 2-3. Parse the original catalog and log the entry count.
	originalEntries, err := ReadCatalogJSON(originalData)
	if err != nil {
		t.Fatalf("ReadCatalogJSON(original): %v", err)
	}
	t.Logf("Original catalog: %d entries", len(originalEntries))

	// 4. Split into per-model YAML files.
	tmpDir := t.TempDir()
	providersDir := filepath.Join(tmpDir, "providers")
	distDir := filepath.Join(tmpDir, "dist")

	if err := Split(originalData, providersDir); err != nil {
		t.Fatalf("Split: %v", err)
	}

	// 5. Count YAML files created.
	yamlFiles, err := filepath.Glob(filepath.Join(providersDir, "*", "models", "*.yaml"))
	if err != nil {
		t.Fatalf("glob YAML files: %v", err)
	}
	t.Logf("YAML files created: %d", len(yamlFiles))

	// 6. Rebuild catalog from the YAML files.
	if err := Build(providersDir, distDir); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// 7. Read the rebuilt catalog.
	rebuiltData, err := os.ReadFile(filepath.Clean(filepath.Join(distDir, "catalog.json")))
	if err != nil {
		t.Fatalf("read rebuilt catalog: %v", err)
	}

	// 8. Parse rebuilt catalog and verify entry count matches.
	rebuiltEntries, err := ReadCatalogJSON(rebuiltData)
	if err != nil {
		t.Fatalf("ReadCatalogJSON(rebuilt): %v", err)
	}
	t.Logf("Rebuilt catalog: %d entries", len(rebuiltEntries))

	if len(originalEntries) != len(rebuiltEntries) {
		t.Fatalf("entry count mismatch: original=%d rebuilt=%d", len(originalEntries), len(rebuiltEntries))
	}

	// 9. Normalize both sides through WriteCatalogJSON (sorts keys) and compare bytes.
	originalNormalized, err := WriteCatalogJSON(originalEntries)
	if err != nil {
		t.Fatalf("WriteCatalogJSON(original): %v", err)
	}

	rebuiltNormalized, err := WriteCatalogJSON(rebuiltEntries)
	if err != nil {
		t.Fatalf("WriteCatalogJSON(rebuilt): %v", err)
	}

	if string(originalNormalized) != string(rebuiltNormalized) {
		origPath := filepath.Join(t.TempDir(), "original_normalized.json")
		rebuildPath := filepath.Join(t.TempDir(), "rebuilt_normalized.json")
		_ = os.WriteFile(origPath, originalNormalized, 0o600)
		_ = os.WriteFile(rebuildPath, rebuiltNormalized, 0o600)
		t.Fatalf("Round-trip FAILED: normalized outputs differ.\n"+
			"Debug files written to:\n"+
			"  %s\n"+
			"  %s\n"+
			"Run: diff %s %s | head -100",
			origPath, rebuildPath, origPath, rebuildPath)
	}

	// 10. Success.
	t.Log("Round-trip PASSED")
}

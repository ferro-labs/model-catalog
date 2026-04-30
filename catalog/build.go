package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Build reads per-model YAML files from providersDir, resolves extends
// inheritance, and writes the catalog JSON, per-provider slices, and manifest
// to distDir.
func Build(providersDir, distDir string) error {
	entries := make(map[string]Entry)

	pattern := filepath.Join(providersDir, "*", "models", "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob %s: %w", pattern, err)
	}

	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		entry, err := ReadModelYAML(data)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		key := entry.Provider + "/" + entry.ModelID
		entries[key] = entry
	}

	entries, err = ResolveExtends(entries)
	if err != nil {
		return fmt.Errorf("resolve extends: %w", err)
	}

	jsonData, err := WriteCatalogJSON(entries)
	if err != nil {
		return fmt.Errorf("write catalog JSON: %w", err)
	}

	if err := os.MkdirAll(distDir, 0o755); err != nil {
		return fmt.Errorf("create dist dir: %w", err)
	}

	outputPath := filepath.Join(distDir, "catalog.json")
	if err := os.WriteFile(outputPath, jsonData, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outputPath, err)
	}

	fmt.Printf("Built catalog with %d entries at %s\n", len(entries), outputPath)

	// Generate per-provider slices and manifest
	if err := generateProviderSlicesAndManifest(entries, jsonData, distDir); err != nil {
		return err
	}

	return nil
}

// sha256Hex returns the lowercase hex SHA-256 digest of data.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// groupByProvider partitions entries by their Provider field.
func groupByProvider(entries map[string]Entry) map[string]map[string]Entry {
	groups := make(map[string]map[string]Entry)
	for key, entry := range entries {
		providerID := entry.Provider
		if groups[providerID] == nil {
			groups[providerID] = make(map[string]Entry)
		}
		groups[providerID][key] = entry
	}
	return groups
}

// generateProviderSlicesAndManifest writes per-provider JSON slices to
// dist/providers/<id>.json and a manifest to dist/manifest.json.
func generateProviderSlicesAndManifest(entries map[string]Entry, catalogJSON []byte, distDir string) error {
	providersDir := filepath.Join(distDir, "providers")
	if err := os.MkdirAll(providersDir, 0o755); err != nil {
		return fmt.Errorf("create providers dir: %w", err)
	}

	catalogHash := sha256Hex(catalogJSON)
	groups := groupByProvider(entries)

	// Collect sorted provider IDs for deterministic output
	providerIDs := make([]string, 0, len(groups))
	for id := range groups {
		providerIDs = append(providerIDs, id)
	}
	sort.Strings(providerIDs)

	manifestProviders := make([]ManifestProvider, 0, len(providerIDs))

	for _, id := range providerIDs {
		sliceEntries := groups[id]

		sliceJSON, err := WriteCatalogJSON(sliceEntries)
		if err != nil {
			return fmt.Errorf("write provider slice %s: %w", id, err)
		}

		slicePath := filepath.Join(providersDir, id+".json")
		if err := os.WriteFile(slicePath, sliceJSON, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", slicePath, err)
		}

		manifestProviders = append(manifestProviders, ManifestProvider{
			ID:         id,
			ModelCount: len(sliceEntries),
			SHA256:     sha256Hex(sliceJSON),
		})
	}

	now := time.Now().UTC()
	manifest := Manifest{
		Version:       fmt.Sprintf("v%d.%02d.%02d", now.Year(), now.Month(), now.Day()),
		SchemaVersion: 1,
		GeneratedAt:   now.Format(time.RFC3339),
		CatalogSHA256: catalogHash,
		Providers:     manifestProviders,
		Stats: ManifestStats{
			TotalModels:    len(entries),
			TotalProviders: len(providerIDs),
		},
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	manifestJSON = append(manifestJSON, '\n')

	manifestPath := filepath.Join(distDir, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestJSON, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", manifestPath, err)
	}

	fmt.Printf("Generated %d provider slices and manifest at %s\n", len(providerIDs), manifestPath)
	return nil
}

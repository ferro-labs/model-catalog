package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Build reads per-model YAML files from providersDir, resolves extends
// inheritance, and writes the catalog JSON, per-provider slices, and manifest
// to distDir.
func Build(providersDir, distDir string) error {
	return BuildWithVersion(providersDir, distDir, "")
}

// BuildWithVersion is like Build, but writes the supplied version into the
// manifest. If version is empty, it derives the CalVer version from UTC now.
func BuildWithVersion(providersDir, distDir, version string) error {
	return buildWithVersionAndGitSHA(providersDir, distDir, version, os.Getenv("GITHUB_SHA"))
}

func buildWithVersionAndGitSHA(providersDir, distDir, version, gitSHA string) error {
	version = strings.TrimSpace(version)
	gitSHA = strings.TrimSpace(gitSHA)
	if version != "" && !strings.HasPrefix(version, "v") {
		return fmt.Errorf("version %q must start with v", version)
	}

	entries := make(map[string]Entry)
	seenPaths := make(map[string]string)

	providerMetas := readProviderMetas(providersDir)

	pattern := filepath.Join(providersDir, "*", "models", "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob %s: %w", pattern, err)
	}

	for _, path := range matches {
		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		entry, err := ReadModelYAML(data)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		key := entry.Provider + "/" + entry.ModelID
		if previousPath, exists := seenPaths[key]; exists {
			return fmt.Errorf("duplicate catalog key %q in %s and %s", key, previousPath, path)
		}
		seenPaths[key] = path
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

	if err := os.MkdirAll(distDir, 0o750); err != nil {
		return fmt.Errorf("create dist dir: %w", err)
	}

	outputPath := filepath.Join(distDir, "catalog.json")
	if err := os.WriteFile(outputPath, jsonData, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", outputPath, err)
	}

	fmt.Printf("Built catalog with %d entries at %s\n", len(entries), outputPath)

	if err := generateProviderSlicesAndManifest(entries, jsonData, distDir, providerMetas, version, gitSHA); err != nil {
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
func readProviderMetas(providersDir string) map[string]ProviderMeta {
	metas := make(map[string]ProviderMeta)
	pattern := filepath.Join(providersDir, "*", "provider.yaml")
	matches, _ := filepath.Glob(pattern)
	for _, path := range matches {
		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			continue
		}
		var meta ProviderMeta
		if err := yaml.Unmarshal(data, &meta); err != nil {
			continue
		}
		providerDir := filepath.Base(filepath.Dir(path))
		metas[providerDir] = meta
	}
	return metas
}

func generateProviderSlicesAndManifest(entries map[string]Entry, catalogJSON []byte, distDir string, providerMetas map[string]ProviderMeta, version, gitSHA string) error {
	providersDir := filepath.Join(distDir, "providers")
	if err := os.MkdirAll(providersDir, 0o750); err != nil {
		return fmt.Errorf("create providers dir: %w", err)
	}

	catalogHash := sha256Hex(catalogJSON)
	if err := os.WriteFile(filepath.Join(distDir, catalogHash+".json"), catalogJSON, 0o600); err != nil {
		return fmt.Errorf("write content-addressed catalog: %w", err)
	}
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
		if err := os.WriteFile(slicePath, sliceJSON, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", slicePath, err)
		}

		sliceHash := sha256Hex(sliceJSON)
		contentDir := filepath.Join(providersDir, id)
		if err := os.MkdirAll(contentDir, 0o750); err != nil {
			return fmt.Errorf("create content-addressed provider directory %s: %w", id, err)
		}
		if err := os.WriteFile(filepath.Join(contentDir, sliceHash+".json"), sliceJSON, 0o600); err != nil {
			return fmt.Errorf("write content-addressed provider slice %s: %w", id, err)
		}

		mp := ManifestProvider{
			ID:         id,
			ModelCount: len(sliceEntries),
			SHA256:     sliceHash,
			URL:        fmt.Sprintf("/v1/providers/%s/%s.json", id, sliceHash),
		}
		if meta, ok := providerMetas[id]; ok {
			mp.DisplayName = meta.DisplayName
			mp.LogoURL = meta.LogoURL
			mp.Logo = meta.Logo
			mp.Category = meta.Category
			mp.Description = meta.Description
			mp.CompanyName = meta.CompanyName
		}
		manifestProviders = append(manifestProviders, mp)
	}

	now := time.Now().UTC()
	if version == "" {
		version = fmt.Sprintf("v%d.%02d.%02d", now.Year(), now.Month(), now.Day())
	}

	manifest := Manifest{
		Version:       version,
		SchemaVersion: 1,
		GeneratedAt:   now.Format(time.RFC3339),
		CatalogSHA256: catalogHash,
		CatalogURL:    "/v1/" + catalogHash + ".json",
		GitSHA:        gitSHA,
		Providers:     manifestProviders,
		Stats: ManifestStats{
			TotalModels:    len(entries),
			TotalProviders: len(providerIDs),
		},
	}

	if err := WriteManifest(distDir, manifest); err != nil {
		return err
	}

	fmt.Printf("Generated %d provider slices and manifest at %s\n", len(providerIDs), filepath.Join(distDir, ManifestFilename))
	return nil
}

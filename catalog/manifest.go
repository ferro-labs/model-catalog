package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ManifestFilename is the manifest file name within a dist directory.
const ManifestFilename = "manifest.json"

// ReadManifest reads and parses the manifest from distDir/manifest.json.
func ReadManifest(distDir string) (Manifest, error) {
	path := filepath.Join(distDir, ManifestFilename)
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Manifest{}, fmt.Errorf("read %s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// WriteManifest writes the manifest to distDir/manifest.json using the canonical
// 2-space indent plus trailing newline. This is the single formatting owner for
// the manifest so every writer (build, release-plan) produces byte-identical output.
func WriteManifest(distDir string, m Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')
	path := filepath.Join(distDir, ManifestFilename)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// Manifest describes the build output: full catalog hash, per-provider slices, and stats.
type Manifest struct {
	Version       string             `json:"version"`
	SchemaVersion int                `json:"schema_version"`
	GeneratedAt   string             `json:"generated_at"`
	CatalogSHA256 string             `json:"catalog_sha256"`
	CatalogURL    string             `json:"catalog_url,omitempty"`
	GitSHA        string             `json:"git_sha,omitempty"`
	Providers     []ManifestProvider `json:"providers"`
	Stats         ManifestStats      `json:"stats"`
}

// ManifestProvider holds metadata for a single provider slice.
type ManifestProvider struct {
	ID          string `json:"id"`
	ModelCount  int    `json:"model_count"`
	SHA256      string `json:"sha256"`
	URL         string `json:"url,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	LogoURL     string `json:"logo_url,omitempty"`
	Logo        string `json:"logo,omitempty"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
}

// ProviderMeta holds provider-level metadata read from provider.yaml.
type ProviderMeta struct {
	DisplayName string `yaml:"display_name" json:"display_name"`
	LogoURL     string `yaml:"logo_url" json:"logo_url"`
	Logo        string `yaml:"logo" json:"logo"`
	Category    string `yaml:"category" json:"category"`
	Description string `yaml:"description" json:"description"`
	CompanyName string `yaml:"company_name" json:"company_name"`
}

// ManifestStats holds aggregate counts.
type ManifestStats struct {
	TotalModels    int `json:"total_models"`
	TotalProviders int `json:"total_providers"`
}

package catalog

// Manifest describes the build output: full catalog hash, per-provider slices, and stats.
type Manifest struct {
	Version       string             `json:"version"`
	SchemaVersion int                `json:"schema_version"`
	GeneratedAt   string             `json:"generated_at"`
	CatalogSHA256 string             `json:"catalog_sha256"`
	Providers     []ManifestProvider `json:"providers"`
	Stats         ManifestStats      `json:"stats"`
}

// ManifestProvider holds metadata for a single provider slice.
type ManifestProvider struct {
	ID         string `json:"id"`
	ModelCount int    `json:"model_count"`
	SHA256     string `json:"sha256"`
}

// ManifestStats holds aggregate counts.
type ManifestStats struct {
	TotalModels    int `json:"total_models"`
	TotalProviders int `json:"total_providers"`
}

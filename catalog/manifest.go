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
	ID          string `json:"id"`
	ModelCount  int    `json:"model_count"`
	SHA256      string `json:"sha256"`
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

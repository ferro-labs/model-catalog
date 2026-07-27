package catalog

import (
	"fmt"
	"strings"
)

// VerifyHash checks that data hashes to the expected lowercase-hex SHA-256.
// A consumer uses this to confirm a fetched artifact matches the manifest
// before trusting its contents. Fails closed when no hash is expected.
func VerifyHash(data []byte, wantSHA256 string) error {
	if strings.TrimSpace(wantSHA256) == "" {
		return fmt.Errorf("no expected sha256")
	}
	got := sha256Hex(data)
	if !strings.EqualFold(got, wantSHA256) {
		return fmt.Errorf("sha256 mismatch: got %s, want %s", got, wantSHA256)
	}
	return nil
}

// VerifyCatalog checks catalog.json bytes against the manifest's catalog_sha256.
func (m Manifest) VerifyCatalog(catalogBytes []byte) error {
	if err := VerifyHash(catalogBytes, m.CatalogSHA256); err != nil {
		return fmt.Errorf("catalog.json: %w", err)
	}
	return nil
}

// VerifyProviderSlice checks a provider slice against its manifest sha256.
// Returns an error if the provider is absent from the manifest so an unknown
// slice can never pass verification.
func (m Manifest) VerifyProviderSlice(id string, data []byte) error {
	for _, p := range m.Providers {
		if p.ID == id {
			if err := VerifyHash(data, p.SHA256); err != nil {
				return fmt.Errorf("provider %s: %w", id, err)
			}
			return nil
		}
	}
	return fmt.Errorf("provider %s: not in manifest", id)
}

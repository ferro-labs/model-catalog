package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultSnapshotBranch is the git branch where archived provenance snapshots
// live, kept out of main's history. Its name is recorded in
// Provenance.SnapshotBranch so a verifier knows where to fetch the content.
const DefaultSnapshotBranch = "snapshots"

// PriceSnapshot is the canonical, content-addressed record of the price-bearing
// data a scraper observed for one model at one point in time. Its SHA-256 is
// stored in Provenance.SnapshotSHA256, letting anyone re-fetch the snapshot and
// confirm the recorded pricing was actually seen at the cited source.
//
// ponytail: for API scrapers this normalized record IS the snapshot; archiving
// raw upstream HTML for Tier 3+ HTML scrapers is the upgrade path.
type PriceSnapshot struct {
	Provider      string   `json:"provider"`
	ModelID       string   `json:"model_id"`
	SourceURL     string   `json:"source_url"`
	VerifiedBy    string   `json:"verified_by"`
	VerifiedAt    string   `json:"verified_at"`
	InputPerM     *float64 `json:"input_per_m_tokens,omitempty"`
	OutputPerM    *float64 `json:"output_per_m_tokens,omitempty"`
	CacheReadPerM *float64 `json:"cache_read_per_m_tokens,omitempty"`
}

// CanonicalJSON renders the snapshot deterministically. json.Marshal emits
// struct fields in declaration order, so the same observation always produces
// the same bytes — and therefore the same hash.
func (s PriceSnapshot) CanonicalJSON() ([]byte, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal snapshot: %w", err)
	}
	return b, nil
}

// SnapshotSHA256 returns the hex-encoded SHA-256 of content.
func SnapshotSHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// SnapshotStore is a content-addressed directory of provenance snapshots.
// Files are named <sha256>.json and writes are idempotent: identical content
// deduplicates to the same file by construction.
type SnapshotStore struct {
	Dir string
}

// Put writes content addressed by its SHA-256 and returns the hex hash. A file
// that already exists is left untouched.
func (s SnapshotStore) Put(content []byte) (string, error) {
	sum := SnapshotSHA256(content)
	if err := os.MkdirAll(s.Dir, 0o750); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", s.Dir, err)
	}
	path := filepath.Join(s.Dir, sum+".json")
	if _, err := os.Stat(path); err == nil {
		return sum, nil // already stored
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return "", fmt.Errorf("write snapshot %s: %w", path, err)
	}
	return sum, nil
}

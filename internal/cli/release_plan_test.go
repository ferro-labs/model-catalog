package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/ferro-labs/model-catalog/catalog"
)

func writeTestManifest(t *testing.T, distDir, version, hash string) {
	t.Helper()
	m := catalog.Manifest{
		Version:       version,
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-01T00:00:00Z",
		CatalogSHA256: hash,
	}
	if err := catalog.WriteManifest(distDir, m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
}

func TestRunReleasePlan_FirstRelease(t *testing.T) {
	dist := t.TempDir()
	writeTestManifest(t, dist, "v2026.06.08", "hashA")

	none := func(distDir, base string) ([]catalog.ReleasedVersion, error) { return nil, nil }

	var out bytes.Buffer
	if err := runReleasePlan(dist, none, &out); err != nil {
		t.Fatalf("runReleasePlan: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "skip=false") || !strings.Contains(got, "version=v2026.06.08") {
		t.Fatalf("output = %q, want skip=false + version=v2026.06.08", got)
	}
}

func TestRunReleasePlan_SkipIdentical(t *testing.T) {
	dist := t.TempDir()
	writeTestManifest(t, dist, "v2026.06.08", "hashA")

	existing := func(distDir, base string) ([]catalog.ReleasedVersion, error) {
		return []catalog.ReleasedVersion{{Version: "v2026.06.08", Hash: "hashA"}}, nil
	}

	var out bytes.Buffer
	if err := runReleasePlan(dist, existing, &out); err != nil {
		t.Fatalf("runReleasePlan: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "skip=true") {
		t.Fatalf("output = %q, want skip=true", got)
	}
	// Manifest version must be untouched on skip.
	m, err := catalog.ReadManifest(dist)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if m.Version != "v2026.06.08" {
		t.Fatalf("version rewritten on skip: %q", m.Version)
	}
}

func TestRunReleasePlan_BumpAndRewriteManifest(t *testing.T) {
	dist := t.TempDir()
	writeTestManifest(t, dist, "v2026.06.08", "hashB")

	existing := func(distDir, base string) ([]catalog.ReleasedVersion, error) {
		return []catalog.ReleasedVersion{{Version: "v2026.06.08", Hash: "hashA"}}, nil
	}

	var out bytes.Buffer
	if err := runReleasePlan(dist, existing, &out); err != nil {
		t.Fatalf("runReleasePlan: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "skip=false") || !strings.Contains(got, "version=v2026.06.08.1") {
		t.Fatalf("output = %q, want version=v2026.06.08.1", got)
	}
	// The released asset's manifest must match the bumped tag.
	m, err := catalog.ReadManifest(dist)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if m.Version != "v2026.06.08.1" {
		t.Fatalf("manifest not rewritten: %q, want v2026.06.08.1", m.Version)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("write failed") }

func TestRunReleasePlan_OutputWriteErrorPropagates(t *testing.T) {
	dist := t.TempDir()
	writeTestManifest(t, dist, "v2026.06.08", "hashA")
	none := func(distDir, base string) ([]catalog.ReleasedVersion, error) { return nil, nil }

	if err := runReleasePlan(dist, none, failingWriter{}); err == nil {
		t.Fatal("expected error when the output write fails, got nil")
	}
}

func TestRunReleasePlan_MissingHashIsFatal(t *testing.T) {
	dist := t.TempDir()
	writeTestManifest(t, dist, "v2026.06.08", "")

	none := func(distDir, base string) ([]catalog.ReleasedVersion, error) { return nil, nil }

	var out bytes.Buffer
	if err := runReleasePlan(dist, none, &out); err == nil {
		t.Fatal("expected error for missing catalog_sha256, got nil")
	}
}

package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotSHA256Deterministic(t *testing.T) {
	in := 3.0
	out := 15.0
	snap := PriceSnapshot{
		Provider: "openai", ModelID: "gpt-5", SourceURL: "https://x",
		VerifiedBy: "scraper-openrouter", VerifiedAt: "2026-08-07",
		InputPerM: &in, OutputPerM: &out,
	}
	a, err := snap.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := snap.CanonicalJSON()
	if SnapshotSHA256(a) != SnapshotSHA256(b) {
		t.Fatal("same snapshot hashed differently")
	}
	// A field change must change the hash.
	snap2 := snap
	other := 4.0
	snap2.InputPerM = &other
	c, _ := snap2.CanonicalJSON()
	if SnapshotSHA256(a) == SnapshotSHA256(c) {
		t.Fatal("different pricing produced the same hash")
	}
}

func TestSnapshotStorePutIdempotent(t *testing.T) {
	dir := t.TempDir()
	store := SnapshotStore{Dir: filepath.Join(dir, "snapshots")}
	content := []byte(`{"a":1}`)

	sha1, err := store.Put(content)
	if err != nil {
		t.Fatal(err)
	}
	if sha1 != SnapshotSHA256(content) {
		t.Fatalf("Put returned %s, want %s", sha1, SnapshotSHA256(content))
	}
	path := filepath.Join(store.Dir, sha1+".json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("snapshot file not written: %v", err)
	}

	// Second Put with identical content is a no-op returning the same hash.
	sha2, err := store.Put(content)
	if err != nil {
		t.Fatal(err)
	}
	if sha1 != sha2 {
		t.Fatalf("idempotent Put returned different hashes: %s vs %s", sha1, sha2)
	}
}

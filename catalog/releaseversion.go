package catalog

import "fmt"

// ReleaseAction is the outcome of resolving a release version.
type ReleaseAction int

const (
	// ActionRelease means a new tag should be created with Decision.Version.
	ActionRelease ReleaseAction = iota
	// ActionSkip means the identical catalog is already published; do nothing.
	ActionSkip
)

// ReleasedVersion is an already-published release in the same CalVer date
// family, paired with the catalog content hash it was built from.
type ReleasedVersion struct {
	Version string // e.g. "v2026.06.08" or "v2026.06.08.1"
	Hash    string // that release's catalog_sha256
}

// Decision is the resolved release plan.
type Decision struct {
	Action  ReleaseAction
	Version string // final version to tag; empty when Action == ActionSkip
}

// ResolveReleaseVersion decides the final release version for a freshly built
// catalog. base is the proposed CalVer name (e.g. "v2026.06.08"), contentHash
// is the catalog_sha256 of the build, and existing is the set of already-
// released versions in the same date family with their content hashes.
//
// Rules, in order:
//  1. If contentHash already matches an existing release, the identical catalog
//     is already published: skip (idempotent).
//  2. Otherwise assign the smallest free name in the family — base, base.1,
//     base.2, … — that no existing version occupies.
func ResolveReleaseVersion(base, contentHash string, existing []ReleasedVersion) Decision {
	taken := make(map[string]struct{}, len(existing))
	for _, rv := range existing {
		if rv.Hash != "" && rv.Hash == contentHash {
			return Decision{Action: ActionSkip}
		}
		taken[rv.Version] = struct{}{}
	}

	if _, exists := taken[base]; !exists {
		return Decision{Action: ActionRelease, Version: base}
	}
	for n := 1; ; n++ {
		candidate := fmt.Sprintf("%s.%d", base, n)
		if _, exists := taken[candidate]; !exists {
			return Decision{Action: ActionRelease, Version: candidate}
		}
	}
}

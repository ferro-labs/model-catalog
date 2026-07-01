package catalog

import "testing"

func TestResolveReleaseVersion(t *testing.T) {
	tests := []struct {
		name        string
		base        string
		contentHash string
		existing    []ReleasedVersion
		wantAction  ReleaseAction
		wantVersion string
	}{
		{
			name:        "first release of the day: base is free",
			base:        "v2026.06.08",
			contentHash: "hashA",
			existing:    nil,
			wantAction:  ActionRelease,
			wantVersion: "v2026.06.08",
		},
		{
			name:        "base taken by different content: bump to .1",
			base:        "v2026.06.08",
			contentHash: "hashB",
			existing:    []ReleasedVersion{{Version: "v2026.06.08", Hash: "hashA"}},
			wantAction:  ActionRelease,
			wantVersion: "v2026.06.08.1",
		},
		{
			name:        "identical content already released as base: skip",
			base:        "v2026.06.08",
			contentHash: "hashA",
			existing:    []ReleasedVersion{{Version: "v2026.06.08", Hash: "hashA"}},
			wantAction:  ActionSkip,
			wantVersion: "",
		},
		{
			name:        "base and .1 taken by different content: bump to .2",
			base:        "v2026.06.08",
			contentHash: "hashC",
			existing: []ReleasedVersion{
				{Version: "v2026.06.08", Hash: "hashA"},
				{Version: "v2026.06.08.1", Hash: "hashB"},
			},
			wantAction:  ActionRelease,
			wantVersion: "v2026.06.08.2",
		},
		{
			name:        "identical content already released under a suffix: skip",
			base:        "v2026.06.08",
			contentHash: "hashB",
			existing: []ReleasedVersion{
				{Version: "v2026.06.08", Hash: "hashA"},
				{Version: "v2026.06.08.1", Hash: "hashB"},
			},
			wantAction:  ActionSkip,
			wantVersion: "",
		},
		{
			name:        "gap in suffixes: fill the smallest free slot",
			base:        "v2026.06.08",
			contentHash: "hashD",
			existing: []ReleasedVersion{
				{Version: "v2026.06.08", Hash: "hashA"},
				{Version: "v2026.06.08.2", Hash: "hashC"},
			},
			wantAction:  ActionRelease,
			wantVersion: "v2026.06.08.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveReleaseVersion(tt.base, tt.contentHash, tt.existing)
			if got.Action != tt.wantAction {
				t.Fatalf("Action = %v, want %v", got.Action, tt.wantAction)
			}
			if got.Version != tt.wantVersion {
				t.Fatalf("Version = %q, want %q", got.Version, tt.wantVersion)
			}
		})
	}
}

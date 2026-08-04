package catalog

import (
	"encoding/json"
	"strings"
	"testing"
)

func minimalEntry() Entry {
	return Entry{
		Provider: "openai", ModelID: "gpt-x", DisplayName: "GPT-X", Mode: "chat",
		Lifecycle: Lifecycle{Status: "ga"}, Tier: "standard",
		Source: "https://openai.com/api/pricing/", UpdatedAt: "2026-08-04",
	}
}

func TestSourcesYAMLRoundTrip(t *testing.T) {
	in := minimalEntry()
	in.Sources = &Sources{Pricing: &Provenance{
		URL: "https://openai.com/api/pricing/", VerifiedAt: "2026-08-04",
		Confidence: "low", VerifiedBy: "import:v1",
	}}
	data, err := WriteModelYAML(in)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err := ReadModelYAML(data)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if out.Sources == nil || out.Sources.Pricing == nil {
		t.Fatalf("sources.pricing lost in round-trip: %s", data)
	}
	if got := out.Sources.Pricing.Confidence; got != "low" {
		t.Errorf("confidence = %q, want low", got)
	}
	if out.Sources.Pricing.SnapshotSHA256 != nil {
		t.Errorf("snapshot should be nil when unset")
	}
}

func TestSourcesOmittedFromJSONWhenNil(t *testing.T) {
	data, err := json.Marshal(minimalEntry())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "\"sources\"") {
		t.Errorf("nil sources must be omitted from JSON, got %s", data)
	}
}

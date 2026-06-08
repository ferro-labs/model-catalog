package scrape

import (
	"testing"

	"github.com/ferro-labs/model-catalog/catalog"
)

func TestReconcileNormalizesModelIDsBeforeGrouping(t *testing.T) {
	entries := map[string]catalog.Entry{}
	observations := []Observation{
		{
			Source:   "models_dev",
			Provider: "anthropic",
			ModelID:  "claude-opus-4-8",
		},
		{
			Source:   "openrouter",
			Provider: "anthropic",
			ModelID:  "claude-opus-4.8",
		},
	}

	result := Reconcile(entries, observations)

	if len(result.NewModels) != 1 {
		t.Fatalf("NewModels = %d, want 1", len(result.NewModels))
	}
	if result.NewModels[0] != "anthropic/claude-opus-4-8" {
		t.Fatalf("NewModels[0] = %q, want anthropic/claude-opus-4-8", result.NewModels[0])
	}
}

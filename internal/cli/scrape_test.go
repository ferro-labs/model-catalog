package cli

import (
	"testing"

	"github.com/ferro-labs/model-catalog/scrape"
)

func ptrFloat(v float64) *float64 { return &v }

func TestBuildAutoAddCandidatesNormalizesModelIDsAcrossSources(t *testing.T) {
	candidates := buildAutoAddCandidates([]string{"anthropic/claude-opus-4-8"}, []scrape.Observation{
		{
			Source:     "models_dev",
			Provider:   "anthropic",
			ModelID:    "claude-opus-4-8",
			InputPerM:  ptrFloat(5),
			OutputPerM: ptrFloat(25),
		},
		{
			Source:     "openrouter",
			Provider:   "anthropic",
			ModelID:    "claude-opus-4.8",
			InputPerM:  ptrFloat(5),
			OutputPerM: ptrFloat(25),
		},
	})

	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(candidates))
	}

	got := candidates[0]
	if got.Provider != "anthropic" {
		t.Fatalf("candidate provider = %q, want anthropic", got.Provider)
	}
	if got.ModelID != "claude-opus-4-8" {
		t.Fatalf("candidate model_id = %q, want claude-opus-4-8", got.ModelID)
	}
	if got.InputPerM == nil || *got.InputPerM != 5 {
		t.Fatalf("candidate input price = %v, want 5", got.InputPerM)
	}
	if got.OutputPerM == nil || *got.OutputPerM != 25 {
		t.Fatalf("candidate output price = %v, want 25", got.OutputPerM)
	}
}

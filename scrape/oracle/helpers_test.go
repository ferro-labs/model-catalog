package oracle

import (
	"math"
	"testing"

	"github.com/ferro-labs/model-catalog/scrape"
)

func findObs(obs []scrape.Observation, provider, modelID string) *scrape.Observation {
	for i := range obs {
		if obs[i].Provider == provider && obs[i].ModelID == modelID {
			return &obs[i]
		}
	}
	return nil
}

func assertFloatPtr(t *testing.T, field string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Errorf("%s: expected %f, got nil", field, want)
		return
	}
	if math.Abs(*got-want) > 0.001 {
		t.Errorf("%s: got %f, want %f", field, *got, want)
	}
}

func assertIntPtr(t *testing.T, field string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Errorf("%s: expected %d, got nil", field, want)
		return
	}
	if *got != want {
		t.Errorf("%s: got %d, want %d", field, *got, want)
	}
}

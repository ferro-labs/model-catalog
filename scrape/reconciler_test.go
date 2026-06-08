package scrape

import (
	"testing"
	"time"

	"github.com/ferro-labs/model-catalog/catalog"
)

func ptr(v float64) *float64 { return &v }

func TestReconcileMatching(t *testing.T) {
	entries := map[string]catalog.Entry{
		"openai/gpt-4o": {
			Provider: "openai",
			ModelID:  "gpt-4o",
			Pricing: catalog.Pricing{
				InputPerMTokens:  catalog.NewNullFloat64(2.5),
				OutputPerMTokens: catalog.NewNullFloat64(10.0),
			},
		},
	}

	obs := []Observation{
		{
			Source:     "openrouter",
			ScrapedAt:  time.Now(),
			Provider:   "openai",
			ModelID:    "gpt-4o",
			InputPerM:  ptr(2.5),
			OutputPerM: ptr(10.0),
		},
	}

	result := Reconcile(entries, obs)

	if result.Checked != 1 {
		t.Errorf("Checked = %d, want 1", result.Checked)
	}
	if result.Matches != 1 {
		t.Errorf("Matches = %d, want 1", result.Matches)
	}
	if len(result.Diffs) != 0 {
		t.Errorf("Diffs = %d, want 0", len(result.Diffs))
	}
}

func TestReconcileSingleSourceDiff(t *testing.T) {
	entries := map[string]catalog.Entry{
		"openai/gpt-4o": {
			Provider: "openai",
			ModelID:  "gpt-4o",
			Pricing: catalog.Pricing{
				InputPerMTokens:  catalog.NewNullFloat64(2.5),
				OutputPerMTokens: catalog.NewNullFloat64(10.0),
			},
		},
	}

	obs := []Observation{
		{
			Source:     "openrouter",
			ScrapedAt:  time.Now(),
			Provider:   "openai",
			ModelID:    "gpt-4o",
			InputPerM:  ptr(3.0), // differs from catalog
			OutputPerM: ptr(10.0),
		},
	}

	result := Reconcile(entries, obs)

	if result.Checked != 1 {
		t.Errorf("Checked = %d, want 1", result.Checked)
	}
	if result.Matches != 0 {
		t.Errorf("Matches = %d, want 0", result.Matches)
	}
	if len(result.Diffs) != 1 {
		t.Fatalf("Diffs = %d, want 1", len(result.Diffs))
	}

	d := result.Diffs[0]
	if d.Confidence != ConfidenceMedium {
		t.Errorf("Confidence = %s, want medium", d.Confidence)
	}
	if d.Field != "pricing.input_per_m_tokens" {
		t.Errorf("Field = %s, want pricing.input_per_m_tokens", d.Field)
	}
}

func TestReconcileMultiSourceAgreement(t *testing.T) {
	entries := map[string]catalog.Entry{
		"openai/gpt-4o": {
			Provider: "openai",
			ModelID:  "gpt-4o",
			Pricing: catalog.Pricing{
				InputPerMTokens:  catalog.NewNullFloat64(2.5),
				OutputPerMTokens: catalog.NewNullFloat64(10.0),
			},
		},
	}

	obs := []Observation{
		{
			Source:     "openrouter",
			ScrapedAt:  time.Now(),
			Provider:   "openai",
			ModelID:    "gpt-4o",
			InputPerM:  ptr(3.0),
			OutputPerM: ptr(10.0),
		},
		{
			Source:     "models_dev",
			ScrapedAt:  time.Now(),
			Provider:   "openai",
			ModelID:    "gpt-4o",
			InputPerM:  ptr(3.0),
			OutputPerM: ptr(10.0),
		},
	}

	result := Reconcile(entries, obs)

	if len(result.Diffs) != 1 {
		t.Fatalf("Diffs = %d, want 1", len(result.Diffs))
	}

	d := result.Diffs[0]
	if d.Confidence != ConfidenceHigh {
		t.Errorf("Confidence = %s, want high", d.Confidence)
	}
	if len(d.Sources) != 2 {
		t.Errorf("Sources = %d, want 2", len(d.Sources))
	}
}

func TestReconcileConflict(t *testing.T) {
	entries := map[string]catalog.Entry{
		"openai/gpt-4o": {
			Provider: "openai",
			ModelID:  "gpt-4o",
			Pricing: catalog.Pricing{
				InputPerMTokens:  catalog.NewNullFloat64(2.5),
				OutputPerMTokens: catalog.NewNullFloat64(10.0),
			},
		},
	}

	obs := []Observation{
		{
			Source:     "openrouter",
			ScrapedAt:  time.Now(),
			Provider:   "openai",
			ModelID:    "gpt-4o",
			InputPerM:  ptr(3.0),
			OutputPerM: ptr(10.0),
		},
		{
			Source:     "models_dev",
			ScrapedAt:  time.Now(),
			Provider:   "openai",
			ModelID:    "gpt-4o",
			InputPerM:  ptr(4.0), // disagrees with openrouter
			OutputPerM: ptr(10.0),
		},
	}

	result := Reconcile(entries, obs)

	if len(result.Diffs) != 1 {
		t.Fatalf("Diffs = %d, want 1", len(result.Diffs))
	}

	d := result.Diffs[0]
	if d.Confidence != ConfidenceConflict {
		t.Errorf("Confidence = %s, want conflict", d.Confidence)
	}
}

func TestReconcileNewModels(t *testing.T) {
	entries := map[string]catalog.Entry{
		"openai/gpt-4o": {
			Provider: "openai",
			ModelID:  "gpt-4o",
			Pricing: catalog.Pricing{
				InputPerMTokens:  catalog.NewNullFloat64(2.5),
				OutputPerMTokens: catalog.NewNullFloat64(10.0),
			},
		},
	}

	obs := []Observation{
		{
			Source:     "openrouter",
			ScrapedAt:  time.Now(),
			Provider:   "openai",
			ModelID:    "gpt-4o",
			InputPerM:  ptr(2.5),
			OutputPerM: ptr(10.0),
		},
		{
			Source:     "openrouter",
			ScrapedAt:  time.Now(),
			Provider:   "openai",
			ModelID:    "gpt-5-turbo",
			InputPerM:  ptr(5.0),
			OutputPerM: ptr(20.0),
		},
	}

	result := Reconcile(entries, obs)

	if len(result.NewModels) != 1 {
		t.Fatalf("NewModels = %d, want 1", len(result.NewModels))
	}
	if result.NewModels[0] != "openai/gpt-5-turbo" {
		t.Errorf("NewModels[0] = %s, want openai/gpt-5-turbo", result.NewModels[0])
	}
}

func TestReconcileMissing(t *testing.T) {
	entries := map[string]catalog.Entry{
		"openai/gpt-4o": {
			Provider: "openai",
			ModelID:  "gpt-4o",
			Pricing: catalog.Pricing{
				InputPerMTokens:  catalog.NewNullFloat64(2.5),
				OutputPerMTokens: catalog.NewNullFloat64(10.0),
			},
		},
		"anthropic/claude-opus": {
			Provider: "anthropic",
			ModelID:  "claude-opus",
			Pricing: catalog.Pricing{
				InputPerMTokens:  catalog.NewNullFloat64(15.0),
				OutputPerMTokens: catalog.NewNullFloat64(75.0),
			},
		},
	}

	// Only openai/gpt-4o has observations.
	obs := []Observation{
		{
			Source:     "openrouter",
			ScrapedAt:  time.Now(),
			Provider:   "openai",
			ModelID:    "gpt-4o",
			InputPerM:  ptr(2.5),
			OutputPerM: ptr(10.0),
		},
	}

	result := Reconcile(entries, obs)

	if len(result.Missing) != 1 {
		t.Fatalf("Missing = %d, want 1", len(result.Missing))
	}
	if result.Missing[0] != "anthropic/claude-opus" {
		t.Errorf("Missing[0] = %s, want anthropic/claude-opus", result.Missing[0])
	}
}

func TestReconcileFloatTolerance(t *testing.T) {
	entries := map[string]catalog.Entry{
		"openai/gpt-4o": {
			Provider: "openai",
			ModelID:  "gpt-4o",
			Pricing: catalog.Pricing{
				InputPerMTokens:  catalog.NewNullFloat64(2.5),
				OutputPerMTokens: catalog.NewNullFloat64(10.0),
			},
		},
	}

	obs := []Observation{
		{
			Source:     "openrouter",
			ScrapedAt:  time.Now(),
			Provider:   "openai",
			ModelID:    "gpt-4o",
			InputPerM:  ptr(2.5001),  // within tolerance
			OutputPerM: ptr(10.0009), // within tolerance
		},
	}

	result := Reconcile(entries, obs)

	if result.Matches != 1 {
		t.Errorf("Matches = %d, want 1 (values within tolerance)", result.Matches)
	}
	if len(result.Diffs) != 0 {
		t.Errorf("Diffs = %d, want 0 (values within tolerance)", len(result.Diffs))
	}
}

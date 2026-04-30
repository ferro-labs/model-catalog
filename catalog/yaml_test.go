package catalog

import (
	"testing"
)

func TestWriteAndReadModelYAML(t *testing.T) {
	entry := Entry{
		Provider:        "anthropic",
		ModelID:         "claude-sonnet-4-20250514",
		DisplayName:     "Claude Sonnet 4",
		Mode:            "chat",
		ContextWindow:   200000,
		MaxOutputTokens: 16000,
		Pricing: Pricing{
			InputPerMTokens:      NewNullFloat64(3.0),
			OutputPerMTokens:     NewNullFloat64(15.0),
			CacheReadPerMTokens:  NewNullFloat64(0.3),
			CacheWritePerMTokens: NewNullFloat64(3.75),
			// remaining pricing fields left as zero value (not valid)
		},
		Capabilities: Capabilities{
			Vision:          true,
			FunctionCalling: true,
			JSONMode:        true,
			Streaming:       true,
			PromptCaching:   true,
		},
		Lifecycle: Lifecycle{
			Status: "ga",
		},
		Source:    "anthropic_api",
		UpdatedAt: "2025-05-15",
		Tier:      "flagship",
	}

	data, err := WriteModelYAML(entry)
	if err != nil {
		t.Fatalf("WriteModelYAML failed: %v", err)
	}

	got, err := ReadModelYAML(data)
	if err != nil {
		t.Fatalf("ReadModelYAML failed: %v", err)
	}

	// Verify basic fields
	if got.Provider != entry.Provider {
		t.Errorf("provider: got %q, want %q", got.Provider, entry.Provider)
	}
	if got.ModelID != entry.ModelID {
		t.Errorf("model_id: got %q, want %q", got.ModelID, entry.ModelID)
	}
	if got.DisplayName != entry.DisplayName {
		t.Errorf("display_name: got %q, want %q", got.DisplayName, entry.DisplayName)
	}
	if got.ContextWindow != entry.ContextWindow {
		t.Errorf("context_window: got %d, want %d", got.ContextWindow, entry.ContextWindow)
	}
	if got.MaxOutputTokens != entry.MaxOutputTokens {
		t.Errorf("max_output_tokens: got %d, want %d", got.MaxOutputTokens, entry.MaxOutputTokens)
	}
	if got.Tier != entry.Tier {
		t.Errorf("tier: got %q, want %q", got.Tier, entry.Tier)
	}

	// Verify valid pricing
	if !got.Pricing.InputPerMTokens.Valid {
		t.Fatal("pricing.input_per_m_tokens: expected valid")
	}
	if got.Pricing.InputPerMTokens.Value != 3.0 {
		t.Errorf("pricing.input_per_m_tokens: got %f, want %f", got.Pricing.InputPerMTokens.Value, 3.0)
	}
	if !got.Pricing.OutputPerMTokens.Valid {
		t.Fatal("pricing.output_per_m_tokens: expected valid")
	}
	if got.Pricing.OutputPerMTokens.Value != 15.0 {
		t.Errorf("pricing.output_per_m_tokens: got %f, want %f", got.Pricing.OutputPerMTokens.Value, 15.0)
	}

	// Verify not-valid pricing stays not-valid
	if got.Pricing.ReasoningPerMTokens.Valid {
		t.Errorf("pricing.reasoning_per_m_tokens: expected not valid, got %f", got.Pricing.ReasoningPerMTokens.Value)
	}
	if got.Pricing.ImagePerTile.Valid {
		t.Errorf("pricing.image_per_tile: expected not valid, got %f", got.Pricing.ImagePerTile.Value)
	}

	// Verify capabilities
	if !got.Capabilities.Vision {
		t.Error("capabilities.vision: expected true")
	}
	if !got.Capabilities.FunctionCalling {
		t.Error("capabilities.function_calling: expected true")
	}
	if got.Capabilities.AudioInput {
		t.Error("capabilities.audio_input: expected false")
	}

	// Verify lifecycle
	if got.Lifecycle.Status != "ga" {
		t.Errorf("lifecycle.status: got %q, want %q", got.Lifecycle.Status, "ga")
	}
	if got.Lifecycle.DeprecationDate != nil {
		t.Errorf("lifecycle.deprecation_date: expected nil, got %q", *got.Lifecycle.DeprecationDate)
	}
}

func TestYAMLRoundTripPreservesNulls(t *testing.T) {
	entry := Entry{
		Provider:        "test",
		ModelID:         "test-model",
		DisplayName:     "Test Model",
		Mode:            "chat",
		ContextWindow:   4096,
		MaxOutputTokens: 1024,
		Pricing: Pricing{
			InputPerMTokens: NewNullFloat64(0.0), // explicitly 0.0, must not become not-valid
			// CacheReadPerMTokens left as zero value (not valid), must not become 0.0
		},
		Capabilities: Capabilities{},
		Lifecycle: Lifecycle{
			Status: "ga",
		},
		Source:    "test",
		UpdatedAt: "2025-01-01",
		Tier:      "free",
	}

	data, err := WriteModelYAML(entry)
	if err != nil {
		t.Fatalf("WriteModelYAML failed: %v", err)
	}

	got, err := ReadModelYAML(data)
	if err != nil {
		t.Fatalf("ReadModelYAML failed: %v", err)
	}

	// 0.0 must NOT become not-valid
	if !got.Pricing.InputPerMTokens.Valid {
		t.Fatal("pricing.input_per_m_tokens: was 0.0, became not-valid after round-trip")
	}
	if got.Pricing.InputPerMTokens.Value != 0.0 {
		t.Errorf("pricing.input_per_m_tokens: got %f, want 0.0", got.Pricing.InputPerMTokens.Value)
	}

	// not-valid must NOT become 0.0
	if got.Pricing.CacheReadPerMTokens.Valid {
		t.Errorf("pricing.cache_read_per_m_tokens: was not-valid, became %f after round-trip", got.Pricing.CacheReadPerMTokens.Value)
	}
}

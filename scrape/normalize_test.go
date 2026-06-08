package scrape

import "testing"

func TestNormalizeProviderModelAnthropicFamilyVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "opus dotted minor",
			input: "claude-opus-4.8",
			want:  "claude-opus-4-8",
		},
		{
			name:  "sonnet dotted minor",
			input: "claude-sonnet-4.5",
			want:  "claude-sonnet-4-5",
		},
		{
			name:  "haiku dotted minor",
			input: "claude-haiku-4.5",
			want:  "claude-haiku-4-5",
		},
		{
			name:  "suffix preserved",
			input: "claude-opus-4.8-fast",
			want:  "claude-opus-4-8-fast",
		},
		{
			name:  "already canonical",
			input: "claude-opus-4-8",
			want:  "claude-opus-4-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := NormalizeProviderModel("anthropic", tt.input)
			if got != tt.want {
				t.Fatalf("NormalizeProviderModel() modelID = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeProviderModelAnthropicGenerationVersion(t *testing.T) {
	_, got := NormalizeProviderModel("anthropic", "claude-3.5-haiku")
	if got != "claude-3-5-haiku" {
		t.Fatalf("NormalizeProviderModel() modelID = %q, want claude-3-5-haiku", got)
	}
}

func TestNormalizeProviderModelOnlyAppliesProviderRules(t *testing.T) {
	_, got := NormalizeProviderModel("openrouter", "anthropic/claude-opus-4.8")
	if got != "anthropic/claude-opus-4.8" {
		t.Fatalf("NormalizeProviderModel() modelID = %q, want anthropic/claude-opus-4.8", got)
	}
}

func TestNormalizeObservationsDoesNotMutateInput(t *testing.T) {
	observations := []Observation{{
		Source:   "openrouter",
		Provider: "anthropic",
		ModelID:  "claude-opus-4.8",
	}}

	normalized := NormalizeObservations(observations)
	if normalized[0].ModelID != "claude-opus-4-8" {
		t.Fatalf("normalized ModelID = %q, want claude-opus-4-8", normalized[0].ModelID)
	}
	if observations[0].ModelID != "claude-opus-4.8" {
		t.Fatalf("input was mutated, got ModelID %q", observations[0].ModelID)
	}
}

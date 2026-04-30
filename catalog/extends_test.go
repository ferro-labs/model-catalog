package catalog

import (
	"strings"
	"testing"
)

func TestResolveExtendsBasic(t *testing.T) {
	base := Entry{
		Provider:        "anthropic",
		ModelID:         "claude-sonnet-4-5",
		DisplayName:     "Claude Sonnet 4.5",
		Mode:            "chat",
		ContextWindow:   200000,
		MaxOutputTokens: 16000,
		Pricing: Pricing{
			InputPerMTokens:      NewNullFloat64(3.0),
			OutputPerMTokens:     NewNullFloat64(15.0),
			CacheReadPerMTokens:  NewNullFloat64(0.3),
			CacheWritePerMTokens: NewNullFloat64(3.75),
		},
		Capabilities: Capabilities{
			Vision:          true,
			FunctionCalling: true,
			Streaming:       true,
		},
		Lifecycle: Lifecycle{Status: "ga"},
		Source:    "anthropic_api",
		UpdatedAt: "2025-05-15",
		Tier:      "flagship",
	}

	wrapper := Entry{
		Extends:     "anthropic/claude-sonnet-4-5",
		Provider:    "vertex_ai",
		ModelID:     "claude-sonnet-4-5",
		DisplayName: "Claude Sonnet 4.5 (via Vertex AI)",
		Pricing: Pricing{
			InputPerMTokens:  NewNullFloat64(3.5),  // override input price
			OutputPerMTokens: NewNullFloat64(15.0),  // same as base (wrappers must include all pricing fields)
			CacheReadPerMTokens:  NewNullFloat64(0.3),
			CacheWritePerMTokens: NewNullFloat64(3.75),
		},
		Capabilities: Capabilities{
			Vision:          true,
			FunctionCalling: true,
			Streaming:       true,
		},
	}

	entries := map[string]Entry{
		"anthropic/claude-sonnet-4-5": base,
		"vertex_ai/claude-sonnet-4-5": wrapper,
	}

	resolved, err := ResolveExtends(entries)
	if err != nil {
		t.Fatalf("ResolveExtends: %v", err)
	}

	got, ok := resolved["vertex_ai/claude-sonnet-4-5"]
	if !ok {
		t.Fatal("missing key vertex_ai/claude-sonnet-4-5")
	}

	// Wrapper's input price should override.
	if !got.Pricing.InputPerMTokens.Valid || got.Pricing.InputPerMTokens.Value != 3.5 {
		t.Errorf("input_per_m_tokens: got %v, want 3.5", got.Pricing.InputPerMTokens)
	}

	// Wrapper specifies all pricing fields; output price matches base.
	if !got.Pricing.OutputPerMTokens.Valid || got.Pricing.OutputPerMTokens.Value != 15.0 {
		t.Errorf("output_per_m_tokens: got %v, want 15.0", got.Pricing.OutputPerMTokens)
	}

	// Cache prices should come from wrapper (same values as base).
	if !got.Pricing.CacheReadPerMTokens.Valid || got.Pricing.CacheReadPerMTokens.Value != 0.3 {
		t.Errorf("cache_read_per_m_tokens: got %v, want 0.3", got.Pricing.CacheReadPerMTokens)
	}
	if !got.Pricing.CacheWritePerMTokens.Valid || got.Pricing.CacheWritePerMTokens.Value != 3.75 {
		t.Errorf("cache_write_per_m_tokens: got %v, want 3.75", got.Pricing.CacheWritePerMTokens)
	}

	// Inherited scalars.
	if got.ContextWindow != 200000 {
		t.Errorf("context_window: got %d, want 200000 (inherited)", got.ContextWindow)
	}
	if got.MaxOutputTokens != 16000 {
		t.Errorf("max_output_tokens: got %d, want 16000 (inherited)", got.MaxOutputTokens)
	}
	if got.Mode != "chat" {
		t.Errorf("mode: got %q, want %q (inherited)", got.Mode, "chat")
	}

	// Wrapper overrides.
	if got.Provider != "vertex_ai" {
		t.Errorf("provider: got %q, want %q", got.Provider, "vertex_ai")
	}
	if got.DisplayName != "Claude Sonnet 4.5 (via Vertex AI)" {
		t.Errorf("display_name: got %q, want %q", got.DisplayName, "Claude Sonnet 4.5 (via Vertex AI)")
	}

	// Base should be unchanged.
	baseResolved := resolved["anthropic/claude-sonnet-4-5"]
	if baseResolved.Provider != "anthropic" {
		t.Errorf("base was mutated: provider=%q", baseResolved.Provider)
	}
}

func TestResolveExtendsProviderRequired(t *testing.T) {
	entries := map[string]Entry{
		"anthropic/claude-sonnet-4-5": {
			Provider:    "anthropic",
			ModelID:     "claude-sonnet-4-5",
			Mode:        "chat",
			DisplayName: "Claude Sonnet 4.5",
		},
		"vertex_ai/claude-sonnet-4-5": {
			Extends: "anthropic/claude-sonnet-4-5",
			// Provider is missing.
			ModelID: "claude-sonnet-4-5",
		},
	}

	_, err := ResolveExtends(entries)
	if err == nil {
		t.Fatal("expected error for missing provider, got nil")
	}
	if !strings.Contains(err.Error(), "missing provider or model_id") {
		t.Errorf("error message does not mention missing fields: %v", err)
	}

	// Also test missing model_id.
	entries2 := map[string]Entry{
		"anthropic/claude-sonnet-4-5": {
			Provider:    "anthropic",
			ModelID:     "claude-sonnet-4-5",
			Mode:        "chat",
			DisplayName: "Claude Sonnet 4.5",
		},
		"vertex_ai/claude-sonnet-4-5": {
			Extends:  "anthropic/claude-sonnet-4-5",
			Provider: "vertex_ai",
			// ModelID is missing.
		},
	}

	_, err = ResolveExtends(entries2)
	if err == nil {
		t.Fatal("expected error for missing model_id, got nil")
	}
	if !strings.Contains(err.Error(), "missing provider or model_id") {
		t.Errorf("error message does not mention missing fields: %v", err)
	}
}

func TestResolveExtendsMaxDepth(t *testing.T) {
	entries := map[string]Entry{
		"base/model-c": {
			Provider:    "base",
			ModelID:     "model-c",
			Mode:        "chat",
			DisplayName: "Model C",
		},
		"mid/model-b": {
			Extends:     "base/model-c",
			Provider:    "mid",
			ModelID:     "model-b",
			DisplayName: "Model B",
		},
		"top/model-a": {
			Extends:     "mid/model-b",
			Provider:    "top",
			ModelID:     "model-a",
			DisplayName: "Model A",
		},
	}

	_, err := ResolveExtends(entries)
	if err == nil {
		t.Fatal("expected error for chain depth > 1, got nil")
	}
	if !strings.Contains(err.Error(), "max chain depth is 1") {
		t.Errorf("error message does not mention max chain depth: %v", err)
	}
}

func TestResolveExtendsModeCantChange(t *testing.T) {
	entries := map[string]Entry{
		"anthropic/claude-sonnet-4-5": {
			Provider:    "anthropic",
			ModelID:     "claude-sonnet-4-5",
			Mode:        "chat",
			DisplayName: "Claude Sonnet 4.5",
		},
		"vertex_ai/claude-sonnet-4-5": {
			Extends:     "anthropic/claude-sonnet-4-5",
			Provider:    "vertex_ai",
			ModelID:     "claude-sonnet-4-5",
			DisplayName: "Claude Sonnet 4.5 (Vertex)",
			Mode:        "embedding", // different from base
		},
	}

	_, err := ResolveExtends(entries)
	if err == nil {
		t.Fatal("expected error for mode change, got nil")
	}
	if !strings.Contains(err.Error(), "mode cannot be overridden") {
		t.Errorf("error message does not mention mode override: %v", err)
	}
}

func TestResolveExtendsBaseNotFound(t *testing.T) {
	entries := map[string]Entry{
		"vertex_ai/claude-sonnet-4-5": {
			Extends:     "anthropic/nonexistent-model",
			Provider:    "vertex_ai",
			ModelID:     "claude-sonnet-4-5",
			DisplayName: "Claude Sonnet 4.5 (Vertex)",
		},
	}

	_, err := ResolveExtends(entries)
	if err == nil {
		t.Fatal("expected error for missing base, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error message does not mention missing base: %v", err)
	}
}

func TestResolveExtendsCapabilitiesOverride(t *testing.T) {
	base := Entry{
		Provider:    "anthropic",
		ModelID:     "claude-sonnet-4-5",
		Mode:        "chat",
		DisplayName: "Claude Sonnet 4.5",
		Capabilities: Capabilities{
			Vision:          true,
			FunctionCalling: true,
			Streaming:       true,
			PromptCaching:   true,
		},
	}

	wrapper := Entry{
		Extends:     "anthropic/claude-sonnet-4-5",
		Provider:    "vertex_ai",
		ModelID:     "claude-sonnet-4-5",
		DisplayName: "Claude Sonnet 4.5 (Vertex)",
		Capabilities: Capabilities{
			Vision:          false, // explicitly false, overrides base
			FunctionCalling: true,
			Streaming:       true,
			PromptCaching:   false, // explicitly false, overrides base
		},
	}

	entries := map[string]Entry{
		"anthropic/claude-sonnet-4-5": base,
		"vertex_ai/claude-sonnet-4-5": wrapper,
	}

	resolved, err := ResolveExtends(entries)
	if err != nil {
		t.Fatalf("ResolveExtends: %v", err)
	}

	got := resolved["vertex_ai/claude-sonnet-4-5"]

	// Wrapper set vision=false, should override base's true.
	if got.Capabilities.Vision {
		t.Error("capabilities.vision: expected false (wrapper override), got true")
	}

	// Wrapper set prompt_caching=false, should override base's true.
	if got.Capabilities.PromptCaching {
		t.Error("capabilities.prompt_caching: expected false (wrapper override), got true")
	}

	// Wrapper set function_calling=true.
	if !got.Capabilities.FunctionCalling {
		t.Error("capabilities.function_calling: expected true, got false")
	}

	// Wrapper set streaming=true.
	if !got.Capabilities.Streaming {
		t.Error("capabilities.streaming: expected true, got false")
	}
}

func TestResolveExtendsStripsExtendsFromOutput(t *testing.T) {
	entries := map[string]Entry{
		"anthropic/claude-sonnet-4-5": {
			Provider:    "anthropic",
			ModelID:     "claude-sonnet-4-5",
			Mode:        "chat",
			DisplayName: "Claude Sonnet 4.5",
		},
		"vertex_ai/claude-sonnet-4-5": {
			Extends:     "anthropic/claude-sonnet-4-5",
			Provider:    "vertex_ai",
			ModelID:     "claude-sonnet-4-5",
			DisplayName: "Claude Sonnet 4.5 (Vertex)",
			Capabilities: Capabilities{
				Vision: true,
			},
		},
	}

	resolved, err := ResolveExtends(entries)
	if err != nil {
		t.Fatalf("ResolveExtends: %v", err)
	}

	for key, entry := range resolved {
		if entry.Extends != "" {
			t.Errorf("entry %q still has Extends=%q after resolution", key, entry.Extends)
		}
	}
}

func TestResolveExtendsNoOp(t *testing.T) {
	// When no entries use extends, output should equal input.
	entries := map[string]Entry{
		"openai/gpt-4o": {
			Provider:    "openai",
			ModelID:     "gpt-4o",
			Mode:        "chat",
			DisplayName: "GPT-4o",
		},
		"anthropic/claude-sonnet-4-5": {
			Provider:    "anthropic",
			ModelID:     "claude-sonnet-4-5",
			Mode:        "chat",
			DisplayName: "Claude Sonnet 4.5",
		},
	}

	resolved, err := ResolveExtends(entries)
	if err != nil {
		t.Fatalf("ResolveExtends: %v", err)
	}

	if len(resolved) != len(entries) {
		t.Fatalf("length mismatch: got %d, want %d", len(resolved), len(entries))
	}

	for key, want := range entries {
		got, ok := resolved[key]
		if !ok {
			t.Errorf("missing key %q", key)
			continue
		}
		if got.Provider != want.Provider || got.ModelID != want.ModelID {
			t.Errorf("entry %q: got provider=%q model_id=%q, want provider=%q model_id=%q",
				key, got.Provider, got.ModelID, want.Provider, want.ModelID)
		}
	}
}

func TestResolveExtendsLifecycleInheritance(t *testing.T) {
	deprecationDate := "2025-12-01"
	sunsetDate := "2026-06-01"
	successor := "anthropic/claude-sonnet-5"

	base := Entry{
		Provider:    "anthropic",
		ModelID:     "claude-sonnet-4-5",
		Mode:        "chat",
		DisplayName: "Claude Sonnet 4.5",
		Lifecycle: Lifecycle{
			Status:          "deprecated",
			DeprecationDate: &deprecationDate,
			SunsetDate:      &sunsetDate,
			Successor:       &successor,
		},
	}

	wrapper := Entry{
		Extends:     "anthropic/claude-sonnet-4-5",
		Provider:    "vertex_ai",
		ModelID:     "claude-sonnet-4-5",
		DisplayName: "Claude Sonnet 4.5 (Vertex)",
		Capabilities: Capabilities{},
		Lifecycle: Lifecycle{
			Status: "ga", // override status only
		},
	}

	entries := map[string]Entry{
		"anthropic/claude-sonnet-4-5": base,
		"vertex_ai/claude-sonnet-4-5": wrapper,
	}

	resolved, err := ResolveExtends(entries)
	if err != nil {
		t.Fatalf("ResolveExtends: %v", err)
	}

	got := resolved["vertex_ai/claude-sonnet-4-5"]

	// Status should be overridden.
	if got.Lifecycle.Status != "ga" {
		t.Errorf("lifecycle.status: got %q, want %q", got.Lifecycle.Status, "ga")
	}

	// Deprecation date, sunset date, successor should be inherited.
	if got.Lifecycle.DeprecationDate == nil || *got.Lifecycle.DeprecationDate != deprecationDate {
		t.Errorf("lifecycle.deprecation_date: expected %q (inherited), got %v", deprecationDate, got.Lifecycle.DeprecationDate)
	}
	if got.Lifecycle.SunsetDate == nil || *got.Lifecycle.SunsetDate != sunsetDate {
		t.Errorf("lifecycle.sunset_date: expected %q (inherited), got %v", sunsetDate, got.Lifecycle.SunsetDate)
	}
	if got.Lifecycle.Successor == nil || *got.Lifecycle.Successor != successor {
		t.Errorf("lifecycle.successor: expected %q (inherited), got %v", successor, got.Lifecycle.Successor)
	}
}

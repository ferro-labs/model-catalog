package catalog

import (
	"strings"
	"testing"
)

// validAgentRouting is a fully-populated AgentRouting block that must pass
// validation. Mirrors the example in the issue body.
func validAgentRouting() *AgentRouting {
	return &AgentRouting{
		CodingQualityTier:    "strong",
		ReasoningQualityTier: "frontier",
		ToolUseQualityTier:   "balanced",
		LatencyTier:          "low",
		LocalSuitability:     "excellent",
		RecommendedRoles:     []string{"planning", "code-review", "implementation"},
	}
}

func validBenchmarks() *Benchmarks {
	return &Benchmarks{
		Coding: &CodingBenchmark{
			Source:    "swe-bench",
			Score:     0.42,
			UpdatedAt: "2026-06-27",
		},
		LocalRuntime: &LocalRuntimeBenchmark{
			Quantization:    "Q4_K_M",
			Backend:         "llama.cpp",
			TokensPerSecond: 71.2,
			Hardware:        "Apple M-series",
		},
	}
}

// TestValidate_AgentRoutingValid ensures a fully-populated escape-routing +
// benchmarks block validates cleanly (acceptance: missing/malformed gating,
// not presence gating).
func TestValidate_AgentRoutingValid(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestModel(t, tmpDir, "openai", "gpt-5-pro.yaml", Entry{
		Provider:     "openai",
		ModelID:      "gpt-5-pro",
		DisplayName:  "GPT-5 Pro",
		Mode:         "chat",
		Lifecycle:    Lifecycle{Status: "ga"},
		Tier:         "flagship",
		AgentRouting: validAgentRouting(),
		Benchmarks:   validBenchmarks(),
		Aliases: Aliases{
			"ferro": {"qwen3.5:397b-cloud", "deepseek-v4-flash:cloud"},
		},
	})

	errs, err := Validate(tmpDir)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected error: %s: %s: %s", e.File, e.Field, e.Message)
		}
	}
}

// TestValidate_AgentRoutingMissingOK ensures models without any
// agent_routing/benchmarks/aliases block still validate (back-compat).
func TestValidate_AgentRoutingMissingOK(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestModel(t, tmpDir, "openai", "gpt-4o.yaml", Entry{
		Provider:    "openai",
		ModelID:     "gpt-4o",
		DisplayName: "GPT-4o",
		Mode:        "chat",
		Lifecycle:   Lifecycle{Status: "ga"},
		Tier:        "flagship",
	})

	errs, err := Validate(tmpDir)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected error: %s: %s: %s", e.File, e.Field, e.Message)
		}
	}
}

// TestValidate_AgentRoutingInvalidEnums ensures malformed enum values are
// rejected across every gated field, while unknown-but-empty ("") values pass.
func TestValidate_AgentRoutingInvalidEnums(t *testing.T) {
	cases := []struct {
		name  string
		field string
		value string
		want  string
	}{
		{"bad coding tier", "agent_routing.coding_quality_tier", "best", "coding_quality_tier"},
		{"bad reasoning tier", "agent_routing.reasoning_quality_tier", "ok", "reasoning_quality_tier"},
		{"bad tool use tier", "agent_routing.tool_use_quality_tier", "great", "tool_use_quality_tier"},
		{"bad latency tier", "agent_routing.latency_tier", "instant", "latency_tier"},
		{"bad local suitability", "agent_routing.local_suitability", "maybe", "local_suitability"},
		{"bad benchmark source", "benchmarks.coding.source", "hearsay", "coding.source"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			entry := Entry{
				Provider:     "openai",
				ModelID:      "gpt-5-pro",
				DisplayName:  "GPT-5 Pro",
				Mode:         "chat",
				Lifecycle:    Lifecycle{Status: "ga"},
				Tier:         "flagship",
				AgentRouting: validAgentRouting(),
				Benchmarks:   validBenchmarks(),
			}
			switch tc.field {
			case "agent_routing.coding_quality_tier":
				entry.AgentRouting.CodingQualityTier = tc.value
			case "agent_routing.reasoning_quality_tier":
				entry.AgentRouting.ReasoningQualityTier = tc.value
			case "agent_routing.tool_use_quality_tier":
				entry.AgentRouting.ToolUseQualityTier = tc.value
			case "agent_routing.latency_tier":
				entry.AgentRouting.LatencyTier = tc.value
			case "agent_routing.local_suitability":
				entry.AgentRouting.LocalSuitability = tc.value
			case "benchmarks.coding.source":
				entry.Benchmarks.Coding.Source = tc.value
			}
			writeTestModel(t, tmpDir, "openai", "gpt-5-pro.yaml", entry)

			errs, err := Validate(tmpDir)
			if err != nil {
				t.Fatalf("Validate() error: %v", err)
			}
			found := false
			for _, e := range errs {
				if strings.Contains(e.Field, tc.field) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected validation error for %s (=%q), got: %+v", tc.field, tc.value, errs)
			}
		})
	}
}

// TestResolveExtends_AgentRoutingMerge ensures a wrapper inherits the base's
// agent_routing block and can override individual tiers without restating the
// entire block. Aliases and benchmarks follow full-replacement semantics.
func TestResolveExtends_AgentRoutingMerge(t *testing.T) {
	base := Entry{
		Provider:     "openai",
		ModelID:      "gpt-5-pro",
		DisplayName:  "GPT-5 Pro",
		Mode:         "chat",
		Lifecycle:    Lifecycle{Status: "ga"},
		Tier:         "flagship",
		AgentRouting: &AgentRouting{
			CodingQualityTier:    "strong",
			ReasoningQualityTier: "frontier",
			ToolUseQualityTier:   "balanced",
			LatencyTier:          "medium",
			LocalSuitability:     "poor",
			RecommendedRoles:     []string{"planning"},
		},
		Aliases:    Aliases{"ferro": {"openai/gpt-5-pro"}},
		Benchmarks: &Benchmarks{Coding: &CodingBenchmark{Source: "swe-bench", Score: 0.42}},
	}
	wrapper := Entry{
		Extends:     "openai/gpt-5-pro",
		Provider:   "openrouter",
		ModelID:    "gpt-5-pro",
		Mode:       "chat",
		AgentRouting: &AgentRouting{
			CodingQualityTier: "frontier", // override
			// ReasoningQualityTier intentionally unset — must inherit from base.
			LatencyTier: "low", // override
		},
		Aliases:    Aliases{"ferro": {"openrouter/gpt-5-pro"}},
		Benchmarks: &Benchmarks{LocalRuntime: &LocalRuntimeBenchmark{Backend: "llama.cpp", TokensPerSecond: 30}},
	}

	entries := map[string]Entry{
		"openai/gpt-5-pro":      base,
		"openrouter/gpt-5-pro":  wrapper,
	}
	resolved, err := ResolveExtends(entries)
	if err != nil {
		t.Fatalf("ResolveExtends() error: %v", err)
	}

	merged := resolved["openrouter/gpt-5-pro"]

	// Wrapper override wins.
	if merged.AgentRouting.CodingQualityTier != "frontier" {
		t.Errorf("CodingQualityTier = %q, want %q (override)", merged.AgentRouting.CodingQualityTier, "frontier")
	}
	if merged.AgentRouting.LatencyTier != "low" {
		t.Errorf("LatencyTier = %q, want %q (override)", merged.AgentRouting.LatencyTier, "low")
	}
	// Unset fields inherit from base.
	if merged.AgentRouting.ReasoningQualityTier != "frontier" {
		t.Errorf("ReasoningQualityTier = %q, want %q (inherited)", merged.AgentRouting.ReasoningQualityTier, "frontier")
	}
	if merged.AgentRouting.ToolUseQualityTier != "balanced" {
		t.Errorf("ToolUseQualityTier = %q, want %q (inherited)", merged.AgentRouting.ToolUseQualityTier, "balanced")
	}
	if merged.AgentRouting.LocalSuitability != "poor" {
		t.Errorf("LocalSuitability = %q, want %q (inherited)", merged.AgentRouting.LocalSuitability, "poor")
	}
	if len(merged.AgentRouting.RecommendedRoles) != 1 || merged.AgentRouting.RecommendedRoles[0] != "planning" {
		t.Errorf("RecommendedRoles = %v, want [planning] (inherited)", merged.AgentRouting.RecommendedRoles)
	}

	// Aliases: full replacement by wrapper.
	if got := merged.Aliases["ferro"]; len(got) != 1 || got[0] != "openrouter/gpt-5-pro" {
		t.Errorf("Aliases[ferro] = %v, want [openrouter/gpt-5-pro] (replaced)", got)
	}

	// Benchmarks: full replacement by wrapper (base Coding block dropped).
	if merged.Benchmarks.Coding != nil {
		t.Errorf("Benchmarks.Coding = %+v, want nil (replaced)", merged.Benchmarks.Coding)
	}
	if merged.Benchmarks.LocalRuntime == nil || merged.Benchmarks.LocalRuntime.Backend != "llama.cpp" {
		t.Errorf("Benchmarks.LocalRuntime = %+v, want backend=llama.cpp (replaced)", merged.Benchmarks.LocalRuntime)
	}

	// Extends must be cleared in output.
	if merged.Extends != "" {
		t.Errorf("Extends = %q, want empty in resolved entry", merged.Extends)
	}
}

// TestEntry_AgentRoutingJSONRoundTrip ensures the new optional fields survive a
// JSON encode/decode cycle with omitempty dropping the block when unset.
func TestEntry_AgentRoutingJSONRoundTrip(t *testing.T) {
	full := Entry{
		Provider:     "openai",
		ModelID:      "gpt-5-pro",
		DisplayName:  "GPT-5 Pro",
		Mode:         "chat",
		Lifecycle:    Lifecycle{Status: "ga"},
		Tier:         "flagship",
		AgentRouting: validAgentRouting(),
		Aliases:      Aliases{"ferro": {"openai/gpt-5-pro"}},
		Benchmarks:   validBenchmarks(),
	}
	data, err := WriteCatalogJSON(map[string]Entry{"openai/gpt-5-pro": full})
	if err != nil {
		t.Fatalf("WriteCatalogJSON: %v", err)
	}

	// The artifact must surface the new top-level keys.
	for _, key := range []string{"agent_routing", "aliases", "benchmarks", "coding_quality_tier", "recommended_roles", "tokens_per_second"} {
		if !strings.Contains(string(data), "\""+key+"\"") {
			t.Errorf("JSON artifact missing key %q\n%s", key, string(data))
		}
	}

	got, err := ReadCatalogJSON(data)
	if err != nil {
		t.Fatalf("ReadCatalogJSON: %v", err)
	}
	round := got["openai/gpt-5-pro"]
	if round.AgentRouting.CodingQualityTier != full.AgentRouting.CodingQualityTier {
		t.Errorf("round-trip CodingQualityTier = %q, want %q", round.AgentRouting.CodingQualityTier, full.AgentRouting.CodingQualityTier)
	}
	if len(round.AgentRouting.RecommendedRoles) != len(full.AgentRouting.RecommendedRoles) {
		t.Errorf("round-trip RecommendedRoles len = %d, want %d", len(round.AgentRouting.RecommendedRoles), len(full.AgentRouting.RecommendedRoles))
	}
	if round.Benchmarks.Coding == nil || round.Benchmarks.Coding.Score != full.Benchmarks.Coding.Score {
		t.Errorf("round-trip Benchmarks.Coding = %+v, want score %v", round.Benchmarks.Coding, full.Benchmarks.Coding.Score)
	}

	// An unset block must be omitted entirely (omitempty).
	empty := Entry{
		Provider:    "openai",
		ModelID:     "gpt-4o",
		DisplayName: "GPT-4o",
		Mode:        "chat",
		Lifecycle:   Lifecycle{Status: "ga"},
		Tier:        "flagship",
	}
	emptyData, err := WriteCatalogJSON(map[string]Entry{"openai/gpt-4o": empty})
	if err != nil {
		t.Fatalf("WriteCatalogJSON (empty): %v", err)
	}
	for _, key := range []string{"\"agent_routing\"", "\"aliases\"", "\"benchmarks\""} {
		if strings.Contains(string(emptyData), key) {
			t.Errorf("empty model JSON should omit %q\n%s", key, string(emptyData))
		}
	}
}

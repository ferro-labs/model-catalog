package catalog

import (
	"testing"
)

const testCatalogJSON = `{
  "anthropic/claude-sonnet-4-20250514": {
    "provider": "anthropic",
    "model_id": "claude-sonnet-4-20250514",
    "display_name": "Claude Sonnet 4",
    "mode": "chat",
    "context_window": 200000,
    "max_output_tokens": 16000,
    "pricing": {
      "input_per_m_tokens": 3.0,
      "output_per_m_tokens": 15.0,
      "cache_read_per_m_tokens": 0.3,
      "cache_write_per_m_tokens": 3.75,
      "reasoning_per_m_tokens": null,
      "image_per_tile": null,
      "audio_input_per_minute": null,
      "audio_output_per_character": null,
      "embedding_per_m_tokens": null,
      "finetune_train_per_m_tokens": null,
      "finetune_input_per_m_tokens": null,
      "finetune_output_per_m_tokens": null
    },
    "capabilities": {
      "vision": true,
      "audio_input": false,
      "audio_output": false,
      "function_calling": true,
      "parallel_tool_calls": true,
      "json_mode": true,
      "response_schema": true,
      "prompt_caching": true,
      "reasoning": false,
      "streaming": true,
      "finetuneable": false
    },
    "lifecycle": {
      "status": "ga",
      "deprecation_date": null,
      "sunset_date": null,
      "successor": null
    },
    "source": "anthropic_api",
    "updated_at": "2025-05-15",
    "tier": "flagship"
  }
}`

func parseTestEntry(t *testing.T) Entry {
	t.Helper()
	catalog, err := ReadCatalogJSON([]byte(testCatalogJSON))
	if err != nil {
		t.Fatalf("ReadCatalogJSON failed: %v", err)
	}
	if len(catalog) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(catalog))
	}
	entry, ok := catalog["anthropic/claude-sonnet-4-20250514"]
	if !ok {
		t.Fatal("expected key 'anthropic/claude-sonnet-4-20250514'")
	}
	return entry
}

func TestReadCatalogJSON_TopLevelFields(t *testing.T) {
	entry := parseTestEntry(t)

	if entry.Provider != "anthropic" {
		t.Errorf("provider: got %q, want %q", entry.Provider, "anthropic")
	}
	if entry.ModelID != "claude-sonnet-4-20250514" {
		t.Errorf("model_id: got %q, want %q", entry.ModelID, "claude-sonnet-4-20250514")
	}
	if entry.DisplayName != "Claude Sonnet 4" {
		t.Errorf("display_name: got %q, want %q", entry.DisplayName, "Claude Sonnet 4")
	}
	if entry.Mode != "chat" {
		t.Errorf("mode: got %q, want %q", entry.Mode, "chat")
	}
	if entry.ContextWindow != 200000 {
		t.Errorf("context_window: got %d, want %d", entry.ContextWindow, 200000)
	}
	if entry.MaxOutputTokens != 16000 {
		t.Errorf("max_output_tokens: got %d, want %d", entry.MaxOutputTokens, 16000)
	}
	if entry.Tier != "flagship" {
		t.Errorf("tier: got %q, want %q", entry.Tier, "flagship")
	}
	if entry.Source != "anthropic_api" {
		t.Errorf("source: got %q, want %q", entry.Source, "anthropic_api")
	}
	if entry.UpdatedAt != "2025-05-15" {
		t.Errorf("updated_at: got %q, want %q", entry.UpdatedAt, "2025-05-15")
	}
}

func TestReadCatalogJSON_Pricing(t *testing.T) {
	entry := parseTestEntry(t)

	if !entry.Pricing.InputPerMTokens.Valid || entry.Pricing.InputPerMTokens.Value != 3.0 {
		t.Errorf("input_per_m_tokens: got {%v, %f}, want {true, 3.0}", entry.Pricing.InputPerMTokens.Valid, entry.Pricing.InputPerMTokens.Value)
	}
	if !entry.Pricing.OutputPerMTokens.Valid || entry.Pricing.OutputPerMTokens.Value != 15.0 {
		t.Errorf("output_per_m_tokens: got {%v, %f}, want {true, 15.0}", entry.Pricing.OutputPerMTokens.Valid, entry.Pricing.OutputPerMTokens.Value)
	}
	if !entry.Pricing.CacheReadPerMTokens.Valid || entry.Pricing.CacheReadPerMTokens.Value != 0.3 {
		t.Errorf("cache_read_per_m_tokens: got {%v, %f}, want {true, 0.3}", entry.Pricing.CacheReadPerMTokens.Valid, entry.Pricing.CacheReadPerMTokens.Value)
	}
	if !entry.Pricing.CacheWritePerMTokens.Valid || entry.Pricing.CacheWritePerMTokens.Value != 3.75 {
		t.Errorf("cache_write_per_m_tokens: got {%v, %f}, want {true, 3.75}", entry.Pricing.CacheWritePerMTokens.Valid, entry.Pricing.CacheWritePerMTokens.Value)
	}
	if entry.Pricing.ReasoningPerMTokens.Valid {
		t.Errorf("reasoning_per_m_tokens: expected not valid")
	}
	if entry.Pricing.ImagePerTile.Valid {
		t.Errorf("image_per_tile: expected not valid")
	}
	if entry.Pricing.AudioInputPerMinute.Valid {
		t.Errorf("audio_input_per_minute: expected not valid")
	}
	if entry.Pricing.EmbeddingPerMTokens.Valid {
		t.Errorf("embedding_per_m_tokens: expected not valid")
	}
}

func TestReadCatalogJSON_Capabilities(t *testing.T) {
	entry := parseTestEntry(t)

	if !entry.Capabilities.Vision {
		t.Error("vision: expected true")
	}
	if entry.Capabilities.AudioInput {
		t.Error("audio_input: expected false")
	}
	if !entry.Capabilities.FunctionCalling {
		t.Error("function_calling: expected true")
	}
	if !entry.Capabilities.ParallelToolCalls {
		t.Error("parallel_tool_calls: expected true")
	}
	if !entry.Capabilities.JSONMode {
		t.Error("json_mode: expected true")
	}
	if !entry.Capabilities.PromptCaching {
		t.Error("prompt_caching: expected true")
	}
	if entry.Capabilities.Reasoning {
		t.Error("reasoning: expected false")
	}
	if !entry.Capabilities.Streaming {
		t.Error("streaming: expected true")
	}
}

func TestReadCatalogJSON_Lifecycle(t *testing.T) {
	entry := parseTestEntry(t)

	if entry.Lifecycle.Status != "ga" {
		t.Errorf("status: got %q, want %q", entry.Lifecycle.Status, "ga")
	}
	if entry.Lifecycle.DeprecationDate != nil {
		t.Errorf("deprecation_date: expected nil, got %q", *entry.Lifecycle.DeprecationDate)
	}
	if entry.Lifecycle.SunsetDate != nil {
		t.Errorf("sunset_date: expected nil, got %q", *entry.Lifecycle.SunsetDate)
	}
	if entry.Lifecycle.Successor != nil {
		t.Errorf("successor: expected nil, got %q", *entry.Lifecycle.Successor)
	}
}

func TestRoundTripJSON(t *testing.T) {
	input := []byte(`{
  "anthropic/claude-sonnet-4-20250514": {
    "provider": "anthropic",
    "model_id": "claude-sonnet-4-20250514",
    "display_name": "Claude Sonnet 4",
    "mode": "chat",
    "context_window": 200000,
    "max_output_tokens": 16000,
    "pricing": {
      "input_per_m_tokens": 3,
      "output_per_m_tokens": 15,
      "cache_read_per_m_tokens": 0.3,
      "cache_write_per_m_tokens": 3.75,
      "reasoning_per_m_tokens": null,
      "image_per_tile": null,
      "audio_input_per_minute": null,
      "audio_output_per_character": null,
      "embedding_per_m_tokens": null,
      "finetune_train_per_m_tokens": null,
      "finetune_input_per_m_tokens": null,
      "finetune_output_per_m_tokens": null
    },
    "capabilities": {
      "vision": true,
      "audio_input": false,
      "audio_output": false,
      "function_calling": true,
      "parallel_tool_calls": true,
      "json_mode": true,
      "response_schema": true,
      "prompt_caching": true,
      "reasoning": false,
      "streaming": true,
      "finetuneable": false
    },
    "lifecycle": {
      "status": "ga",
      "deprecation_date": null,
      "sunset_date": null,
      "successor": null
    },
    "source": "anthropic_api",
    "updated_at": "2025-05-15",
    "tier": "flagship"
  },
  "openai/gpt-4o-2024-11-20": {
    "provider": "openai",
    "model_id": "gpt-4o-2024-11-20",
    "display_name": "GPT-4o",
    "mode": "chat",
    "context_window": 128000,
    "max_output_tokens": 16384,
    "pricing": {
      "input_per_m_tokens": 2.5,
      "output_per_m_tokens": 10,
      "cache_read_per_m_tokens": 1.25,
      "cache_write_per_m_tokens": null,
      "reasoning_per_m_tokens": null,
      "image_per_tile": null,
      "audio_input_per_minute": null,
      "audio_output_per_character": null,
      "embedding_per_m_tokens": null,
      "finetune_train_per_m_tokens": null,
      "finetune_input_per_m_tokens": null,
      "finetune_output_per_m_tokens": null
    },
    "capabilities": {
      "vision": true,
      "audio_input": false,
      "audio_output": false,
      "function_calling": true,
      "parallel_tool_calls": true,
      "json_mode": true,
      "response_schema": true,
      "prompt_caching": true,
      "reasoning": false,
      "streaming": true,
      "finetuneable": true
    },
    "lifecycle": {
      "status": "ga",
      "deprecation_date": null,
      "sunset_date": null,
      "successor": null
    },
    "source": "openai_api",
    "updated_at": "2024-11-20",
    "tier": "flagship"
  }
}
`)

	catalog1, err := ReadCatalogJSON(input)
	if err != nil {
		t.Fatalf("first ReadCatalogJSON failed: %v", err)
	}
	if len(catalog1) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(catalog1))
	}

	output1, err := WriteCatalogJSON(catalog1)
	if err != nil {
		t.Fatalf("first WriteCatalogJSON failed: %v", err)
	}

	catalog2, err := ReadCatalogJSON(output1)
	if err != nil {
		t.Fatalf("second ReadCatalogJSON failed: %v", err)
	}

	output2, err := WriteCatalogJSON(catalog2)
	if err != nil {
		t.Fatalf("second WriteCatalogJSON failed: %v", err)
	}

	if string(output1) != string(output2) {
		t.Errorf("round-trip outputs differ:\n--- first write ---\n%s\n--- second write ---\n%s", string(output1), string(output2))
	}

	anthIdx := -1
	openIdx := -1
	for i := range output1 {
		if i+10 < len(output1) && string(output1[i:i+10]) == "\"anthropic" {
			if anthIdx == -1 {
				anthIdx = i
			}
		}
		if i+7 < len(output1) && string(output1[i:i+7]) == "\"openai" {
			if openIdx == -1 {
				openIdx = i
			}
		}
	}
	if anthIdx >= openIdx {
		t.Errorf("expected anthropic key before openai key, anthIdx=%d openIdx=%d", anthIdx, openIdx)
	}
}

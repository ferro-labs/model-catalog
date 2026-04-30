package catalog

import (
	"testing"
)

func TestReadCatalogJSON(t *testing.T) {
	input := []byte(`{
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
}
`)

	catalog, err := ReadCatalogJSON(input)
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

	// Pricing: valid fields
	if !entry.Pricing.InputPerMTokens.Valid {
		t.Fatal("pricing.input_per_m_tokens: expected valid")
	}
	if entry.Pricing.InputPerMTokens.Value != 3.0 {
		t.Errorf("pricing.input_per_m_tokens: got %f, want %f", entry.Pricing.InputPerMTokens.Value, 3.0)
	}
	if !entry.Pricing.OutputPerMTokens.Valid {
		t.Fatal("pricing.output_per_m_tokens: expected valid")
	}
	if entry.Pricing.OutputPerMTokens.Value != 15.0 {
		t.Errorf("pricing.output_per_m_tokens: got %f, want %f", entry.Pricing.OutputPerMTokens.Value, 15.0)
	}
	if !entry.Pricing.CacheReadPerMTokens.Valid {
		t.Fatal("pricing.cache_read_per_m_tokens: expected valid")
	}
	if entry.Pricing.CacheReadPerMTokens.Value != 0.3 {
		t.Errorf("pricing.cache_read_per_m_tokens: got %f, want %f", entry.Pricing.CacheReadPerMTokens.Value, 0.3)
	}
	if !entry.Pricing.CacheWritePerMTokens.Valid {
		t.Fatal("pricing.cache_write_per_m_tokens: expected valid")
	}
	if entry.Pricing.CacheWritePerMTokens.Value != 3.75 {
		t.Errorf("pricing.cache_write_per_m_tokens: got %f, want %f", entry.Pricing.CacheWritePerMTokens.Value, 3.75)
	}

	// Pricing: null fields
	if entry.Pricing.ReasoningPerMTokens.Valid {
		t.Errorf("pricing.reasoning_per_m_tokens: expected not valid, got %f", entry.Pricing.ReasoningPerMTokens.Value)
	}
	if entry.Pricing.ImagePerTile.Valid {
		t.Errorf("pricing.image_per_tile: expected not valid, got %f", entry.Pricing.ImagePerTile.Value)
	}
	if entry.Pricing.AudioInputPerMinute.Valid {
		t.Errorf("pricing.audio_input_per_minute: expected not valid, got %f", entry.Pricing.AudioInputPerMinute.Value)
	}
	if entry.Pricing.EmbeddingPerMTokens.Valid {
		t.Errorf("pricing.embedding_per_m_tokens: expected not valid, got %f", entry.Pricing.EmbeddingPerMTokens.Value)
	}

	// Capabilities
	if !entry.Capabilities.Vision {
		t.Error("capabilities.vision: expected true")
	}
	if entry.Capabilities.AudioInput {
		t.Error("capabilities.audio_input: expected false")
	}
	if !entry.Capabilities.FunctionCalling {
		t.Error("capabilities.function_calling: expected true")
	}
	if !entry.Capabilities.ParallelToolCalls {
		t.Error("capabilities.parallel_tool_calls: expected true")
	}
	if !entry.Capabilities.JSONMode {
		t.Error("capabilities.json_mode: expected true")
	}
	if !entry.Capabilities.PromptCaching {
		t.Error("capabilities.prompt_caching: expected true")
	}
	if entry.Capabilities.Reasoning {
		t.Error("capabilities.reasoning: expected false")
	}
	if !entry.Capabilities.Streaming {
		t.Error("capabilities.streaming: expected true")
	}

	// Lifecycle
	if entry.Lifecycle.Status != "ga" {
		t.Errorf("lifecycle.status: got %q, want %q", entry.Lifecycle.Status, "ga")
	}
	if entry.Lifecycle.DeprecationDate != nil {
		t.Errorf("lifecycle.deprecation_date: expected nil, got %q", *entry.Lifecycle.DeprecationDate)
	}
	if entry.Lifecycle.SunsetDate != nil {
		t.Errorf("lifecycle.sunset_date: expected nil, got %q", *entry.Lifecycle.SunsetDate)
	}
	if entry.Lifecycle.Successor != nil {
		t.Errorf("lifecycle.successor: expected nil, got %q", *entry.Lifecycle.Successor)
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

	// First parse
	catalog1, err := ReadCatalogJSON(input)
	if err != nil {
		t.Fatalf("first ReadCatalogJSON failed: %v", err)
	}

	if len(catalog1) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(catalog1))
	}

	// First write
	output1, err := WriteCatalogJSON(catalog1)
	if err != nil {
		t.Fatalf("first WriteCatalogJSON failed: %v", err)
	}

	// Second parse
	catalog2, err := ReadCatalogJSON(output1)
	if err != nil {
		t.Fatalf("second ReadCatalogJSON failed: %v", err)
	}

	// Second write
	output2, err := WriteCatalogJSON(catalog2)
	if err != nil {
		t.Fatalf("second WriteCatalogJSON failed: %v", err)
	}

	// Byte-identical check
	if string(output1) != string(output2) {
		t.Errorf("round-trip outputs differ:\n--- first write ---\n%s\n--- second write ---\n%s", string(output1), string(output2))
	}

	// Verify key ordering: anthropic should come before openai
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

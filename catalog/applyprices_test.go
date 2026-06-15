package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const baseModelYAML = `provider: openai
model_id: gpt-x
display_name: gpt-x
mode: chat
context_window: 0
max_output_tokens: 0
pricing:
    input_per_m_tokens: null
    output_per_m_tokens: 1.25
    cache_read_per_m_tokens: 0.3
    cache_write_per_m_tokens: null
    reasoning_per_m_tokens: null
    image_per_tile: null
    audio_input_per_minute: null
    audio_output_per_character: null
    embedding_per_m_tokens: null
    finetune_train_per_m_tokens: null
    finetune_input_per_m_tokens: null
    finetune_output_per_m_tokens: null
capabilities:
    vision: false
    audio_input: false
    audio_output: false
    function_calling: false
    parallel_tool_calls: false
    json_mode: false
    response_schema: false
    prompt_caching: false
    reasoning: false
    streaming: false
    finetuneable: false
lifecycle:
    status: preview
    deprecation_date: null
    sunset_date: null
    successor: null
source: auto:models_dev+openrouter
updated_at: "2026-01-01"
tier: standard
`

const wrapperModelYAML = `extends: openai/gpt-x
provider: azure_openai
model_id: gpt-x
display_name: "GPT-X (Azure)"
pricing:
    input_per_m_tokens: 0.4
    output_per_m_tokens: 1.6
    cache_read_per_m_tokens: 0.1
    cache_write_per_m_tokens: null
    reasoning_per_m_tokens: null
    image_per_tile: null
    audio_input_per_minute: null
    audio_output_per_character: null
    embedding_per_m_tokens: null
    finetune_train_per_m_tokens: null
    finetune_input_per_m_tokens: null
    finetune_output_per_m_tokens: null
capabilities:
    vision: true
    audio_input: false
    audio_output: false
    function_calling: true
    parallel_tool_calls: true
    json_mode: true
    response_schema: true
    prompt_caching: true
    reasoning: false
    streaming: true
    finetuneable: false
lifecycle:
    status: ga
    deprecation_date: null
    sunset_date: null
    successor: null
tier: flagship
`

func writeModel(t *testing.T, root, provider, modelID, content string) string {
	t.Helper()
	dir := filepath.Join(root, provider, "models")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, SanitizeFilename(modelID)+".yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write model: %v", err)
	}
	return path
}

// reload parses a written file back into an Entry for value assertions.
func reload(t *testing.T, path string) Entry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	e, err := ReadModelYAML(data)
	if err != nil {
		t.Fatalf("parse back: %v", err)
	}
	return e
}

func TestApplyPriceUpdates_GapFillNull(t *testing.T) {
	root := t.TempDir()
	path := writeModel(t, root, "openai", "gpt-x", baseModelYAML)

	res, err := ApplyPriceUpdates(root, []PriceUpdate{
		{Provider: "openai", ModelID: "gpt-x", Field: "input_per_m_tokens", Value: 0.2},
	}, "2026-06-15", false)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied != 1 || res.Files != 1 || len(res.NotApplied) != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}

	e := reload(t, path)
	if !e.Pricing.InputPerMTokens.Valid || e.Pricing.InputPerMTokens.Value != 0.2 {
		t.Errorf("input not filled: %+v", e.Pricing.InputPerMTokens)
	}
	// Untouched fields preserved.
	if !e.Pricing.OutputPerMTokens.Valid || e.Pricing.OutputPerMTokens.Value != 1.25 {
		t.Errorf("output changed unexpectedly: %+v", e.Pricing.OutputPerMTokens)
	}
	if e.UpdatedAt != "2026-06-15" {
		t.Errorf("updated_at not bumped: %q", e.UpdatedAt)
	}
}

func TestApplyPriceUpdates_OverwriteExisting(t *testing.T) {
	root := t.TempDir()
	path := writeModel(t, root, "openai", "gpt-x", baseModelYAML)

	res, err := ApplyPriceUpdates(root, []PriceUpdate{
		{Provider: "openai", ModelID: "gpt-x", Field: "cache_read_per_m_tokens", Value: 0.25},
	}, "2026-06-15", false)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied != 1 {
		t.Fatalf("expected 1 applied, got %+v", res)
	}
	e := reload(t, path)
	if !e.Pricing.CacheReadPerMTokens.Valid || e.Pricing.CacheReadPerMTokens.Value != 0.25 {
		t.Errorf("cache_read not overwritten: %+v", e.Pricing.CacheReadPerMTokens)
	}
}

func TestApplyPriceUpdates_PreservesExtends(t *testing.T) {
	root := t.TempDir()
	path := writeModel(t, root, "azure_openai", "gpt-x", wrapperModelYAML)

	_, err := ApplyPriceUpdates(root, []PriceUpdate{
		{Provider: "azure_openai", ModelID: "gpt-x", Field: "input_per_m_tokens", Value: 0.5},
	}, "2026-06-15", false)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "extends: openai/gpt-x") {
		t.Errorf("extends key lost:\n%s", data)
	}
	e := reload(t, path)
	if e.Pricing.InputPerMTokens.Value != 0.5 {
		t.Errorf("input not updated on wrapper: %+v", e.Pricing.InputPerMTokens)
	}
}

func TestApplyPriceUpdates_MissingFileReported(t *testing.T) {
	root := t.TempDir()
	res, err := ApplyPriceUpdates(root, []PriceUpdate{
		{Provider: "nope", ModelID: "ghost", Field: "input_per_m_tokens", Value: 1.0},
	}, "2026-06-15", false)
	if err != nil {
		t.Fatalf("apply should not error on missing file: %v", err)
	}
	if res.Applied != 0 || len(res.NotApplied) != 1 {
		t.Fatalf("expected 1 not-applied, got %+v", res)
	}
}

func TestApplyPriceUpdates_DryRunDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	path := writeModel(t, root, "openai", "gpt-x", baseModelYAML)
	before, _ := os.ReadFile(path)

	res, err := ApplyPriceUpdates(root, []PriceUpdate{
		{Provider: "openai", ModelID: "gpt-x", Field: "input_per_m_tokens", Value: 0.2},
	}, "2026-06-15", true)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied != 1 || res.Files != 1 {
		t.Fatalf("dry-run should count would-be changes: %+v", res)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Errorf("dry-run modified the file")
	}
}

func TestApplyPriceUpdates_MultipleFieldsOneFile(t *testing.T) {
	root := t.TempDir()
	path := writeModel(t, root, "openai", "gpt-x", baseModelYAML)

	res, err := ApplyPriceUpdates(root, []PriceUpdate{
		{Provider: "openai", ModelID: "gpt-x", Field: "input_per_m_tokens", Value: 0.2},
		{Provider: "openai", ModelID: "gpt-x", Field: "output_per_m_tokens", Value: 0.9},
	}, "2026-06-15", false)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied != 2 || res.Files != 1 {
		t.Fatalf("expected 2 applied across 1 file: %+v", res)
	}
	e := reload(t, path)
	if e.Pricing.InputPerMTokens.Value != 0.2 || e.Pricing.OutputPerMTokens.Value != 0.9 {
		t.Errorf("fields not both updated: %+v", e.Pricing)
	}
}

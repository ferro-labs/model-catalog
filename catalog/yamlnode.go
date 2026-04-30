package catalog

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// YAML node helpers for building structured YAML output.

// AddStringField appends a string key-value pair to a YAML mapping node.
func AddStringField(mapping *yaml.Node, key, value string) {
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Value: value}
	// Quote updated_at to preserve date format.
	if key == "updated_at" {
		valNode.Style = yaml.DoubleQuotedStyle
	}
	mapping.Content = append(mapping.Content, keyNode, valNode)
}

// AddIntField appends an integer key-value pair to a YAML mapping node.
func AddIntField(mapping *yaml.Node, key string, value int) {
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", value), Tag: "!!int"}
	mapping.Content = append(mapping.Content, keyNode, valNode)
}

// AddBoolField appends a boolean key-value pair to a YAML mapping node.
func AddBoolField(mapping *yaml.Node, key string, value bool) {
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	v := "false"
	if value {
		v = "true"
	}
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Value: v, Tag: "!!bool"}
	mapping.Content = append(mapping.Content, keyNode, valNode)
}

// AddNullFloat64Field appends a NullFloat64 key-value pair to a YAML mapping node.
func AddNullFloat64Field(mapping *yaml.Node, key string, nf NullFloat64) {
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	var valNode *yaml.Node
	if !nf.Valid {
		valNode = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	} else {
		valNode = &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%g", nf.Value)}
	}
	mapping.Content = append(mapping.Content, keyNode, valNode)
}

// AddPtrStringField appends a *string key-value pair to a YAML mapping node.
func AddPtrStringField(mapping *yaml.Node, key string, value *string) {
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	var valNode *yaml.Node
	if value == nil {
		valNode = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	} else {
		valNode = &yaml.Node{Kind: yaml.ScalarNode, Value: *value}
	}
	mapping.Content = append(mapping.Content, keyNode, valNode)
}

// PricingToYAML converts a Pricing struct to a YAML mapping node with all 12 fields.
func PricingToYAML(p Pricing) *yaml.Node {
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	AddNullFloat64Field(mapping, "input_per_m_tokens", p.InputPerMTokens)
	AddNullFloat64Field(mapping, "output_per_m_tokens", p.OutputPerMTokens)
	AddNullFloat64Field(mapping, "cache_read_per_m_tokens", p.CacheReadPerMTokens)
	AddNullFloat64Field(mapping, "cache_write_per_m_tokens", p.CacheWritePerMTokens)
	AddNullFloat64Field(mapping, "reasoning_per_m_tokens", p.ReasoningPerMTokens)
	AddNullFloat64Field(mapping, "image_per_tile", p.ImagePerTile)
	AddNullFloat64Field(mapping, "audio_input_per_minute", p.AudioInputPerMinute)
	AddNullFloat64Field(mapping, "audio_output_per_character", p.AudioOutputPerCharacter)
	AddNullFloat64Field(mapping, "embedding_per_m_tokens", p.EmbeddingPerMTokens)
	AddNullFloat64Field(mapping, "finetune_train_per_m_tokens", p.FinetuneTrainPerMTokens)
	AddNullFloat64Field(mapping, "finetune_input_per_m_tokens", p.FinetuneInputPerMTokens)
	AddNullFloat64Field(mapping, "finetune_output_per_m_tokens", p.FinetuneOutputPerMTokens)
	return mapping
}

// CapabilitiesToYAML converts a Capabilities struct to a YAML mapping node.
func CapabilitiesToYAML(caps Capabilities) *yaml.Node {
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	AddBoolField(mapping, "vision", caps.Vision)
	AddBoolField(mapping, "audio_input", caps.AudioInput)
	AddBoolField(mapping, "audio_output", caps.AudioOutput)
	AddBoolField(mapping, "function_calling", caps.FunctionCalling)
	AddBoolField(mapping, "parallel_tool_calls", caps.ParallelToolCalls)
	AddBoolField(mapping, "json_mode", caps.JSONMode)
	AddBoolField(mapping, "response_schema", caps.ResponseSchema)
	AddBoolField(mapping, "prompt_caching", caps.PromptCaching)
	AddBoolField(mapping, "reasoning", caps.Reasoning)
	AddBoolField(mapping, "streaming", caps.Streaming)
	AddBoolField(mapping, "finetuneable", caps.Finetuneable)
	return mapping
}

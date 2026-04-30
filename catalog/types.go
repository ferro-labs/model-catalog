package catalog

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// NullFloat64 is a nullable float64 that serializes as null in JSON when not valid,
// and always includes at least one decimal place for whole numbers (3.0, not 3).
type NullFloat64 struct {
	Value float64
	Valid bool
}

// NewNullFloat64 creates a valid NullFloat64 with the given value.
func NewNullFloat64(v float64) NullFloat64 {
	return NullFloat64{Value: v, Valid: true}
}

func (f NullFloat64) MarshalJSON() ([]byte, error) {
	if !f.Valid {
		return []byte("null"), nil
	}
	s := strconv.FormatFloat(f.Value, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return []byte(s), nil
}

func (f *NullFloat64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		f.Valid = false
		return nil
	}
	val, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return err
	}
	f.Value = val
	f.Valid = true
	return nil
}

func (f NullFloat64) MarshalYAML() (any, error) {
	if !f.Valid {
		return nil, nil
	}
	return f.Value, nil
}

func (f *NullFloat64) UnmarshalYAML(value *yaml.Node) error {
	if value.Tag == "!!null" || value.Value == "null" || value.Value == "~" || value.Value == "" {
		f.Valid = false
		return nil
	}
	val, err := strconv.ParseFloat(value.Value, 64)
	if err != nil {
		return fmt.Errorf("cannot parse %q as float64: %w", value.Value, err)
	}
	f.Value = val
	f.Valid = true
	return nil
}

type Entry struct {
	Extends         string       `json:"-" yaml:"extends,omitempty"`
	Provider        string       `json:"provider" yaml:"provider"`
	ModelID         string       `json:"model_id" yaml:"model_id"`
	DisplayName     string       `json:"display_name" yaml:"display_name"`
	Mode            string       `json:"mode" yaml:"mode"`
	ContextWindow   int          `json:"context_window" yaml:"context_window"`
	MaxOutputTokens int          `json:"max_output_tokens" yaml:"max_output_tokens"`
	Pricing         Pricing      `json:"pricing" yaml:"pricing"`
	Capabilities    Capabilities `json:"capabilities" yaml:"capabilities"`
	Lifecycle       Lifecycle    `json:"lifecycle" yaml:"lifecycle"`
	Source          string       `json:"source" yaml:"source"`
	UpdatedAt       string       `json:"updated_at" yaml:"updated_at"`
	Tier            string       `json:"tier" yaml:"tier"`
}

type Pricing struct {
	InputPerMTokens          NullFloat64 `json:"input_per_m_tokens" yaml:"input_per_m_tokens"`
	OutputPerMTokens         NullFloat64 `json:"output_per_m_tokens" yaml:"output_per_m_tokens"`
	CacheReadPerMTokens      NullFloat64 `json:"cache_read_per_m_tokens" yaml:"cache_read_per_m_tokens"`
	CacheWritePerMTokens     NullFloat64 `json:"cache_write_per_m_tokens" yaml:"cache_write_per_m_tokens"`
	ReasoningPerMTokens      NullFloat64 `json:"reasoning_per_m_tokens" yaml:"reasoning_per_m_tokens"`
	ImagePerTile             NullFloat64 `json:"image_per_tile" yaml:"image_per_tile"`
	AudioInputPerMinute      NullFloat64 `json:"audio_input_per_minute" yaml:"audio_input_per_minute"`
	AudioOutputPerCharacter  NullFloat64 `json:"audio_output_per_character" yaml:"audio_output_per_character"`
	EmbeddingPerMTokens      NullFloat64 `json:"embedding_per_m_tokens" yaml:"embedding_per_m_tokens"`
	FinetuneTrainPerMTokens  NullFloat64 `json:"finetune_train_per_m_tokens" yaml:"finetune_train_per_m_tokens"`
	FinetuneInputPerMTokens  NullFloat64 `json:"finetune_input_per_m_tokens" yaml:"finetune_input_per_m_tokens"`
	FinetuneOutputPerMTokens NullFloat64 `json:"finetune_output_per_m_tokens" yaml:"finetune_output_per_m_tokens"`
}

type Capabilities struct {
	Vision            bool `json:"vision" yaml:"vision"`
	AudioInput        bool `json:"audio_input" yaml:"audio_input"`
	AudioOutput       bool `json:"audio_output" yaml:"audio_output"`
	FunctionCalling   bool `json:"function_calling" yaml:"function_calling"`
	ParallelToolCalls bool `json:"parallel_tool_calls" yaml:"parallel_tool_calls"`
	JSONMode          bool `json:"json_mode" yaml:"json_mode"`
	ResponseSchema    bool `json:"response_schema" yaml:"response_schema"`
	PromptCaching     bool `json:"prompt_caching" yaml:"prompt_caching"`
	Reasoning         bool `json:"reasoning" yaml:"reasoning"`
	Streaming         bool `json:"streaming" yaml:"streaming"`
	Finetuneable      bool `json:"finetuneable" yaml:"finetuneable"`
}

type Lifecycle struct {
	Status          string  `json:"status" yaml:"status"`
	DeprecationDate *string `json:"deprecation_date" yaml:"deprecation_date"`
	SunsetDate      *string `json:"sunset_date" yaml:"sunset_date"`
	Successor       *string `json:"successor" yaml:"successor"`
}

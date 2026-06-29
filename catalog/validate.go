package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidationError represents a single validation failure for a model file.
type ValidationError struct {
	File    string
	Field   string
	Message string
}

var validModes = map[string]bool{
	"chat":      true,
	"embedding": true,
	"image":     true,
	"audio_in":  true,
	"audio_out": true,
}

var validStatuses = map[string]bool{
	"preview":    true,
	"ga":         true,
	"deprecated": true,
	"sunset":     true,
	"legacy":     true,
}

var validTiers = map[string]bool{
	"flagship": true,
	"standard": true,
}

// Agent-routing enum sets. Empty (unset) values are always valid — these
// only gate non-empty malformed values.
var validCodingTiers = map[string]bool{
	"frontier":    true,
	"strong":      true,
	"balanced":    true,
	"fast":        true,
	"experimental": true,
	"unknown":     true,
}

var validToolUseTiers = map[string]bool{
	"strong":  true,
	"balanced": true,
	"weak":    true,
	"unknown": true,
}

var validLatencyTiers = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
	"unknown": true,
}

var validLocalSuitability = map[string]bool{
	"excellent": true,
	"good":      true,
	"poor":      true,
	"unknown":   true,
}

var validCodingBenchmarkSources = map[string]bool{
	"swe-bench": true,
	"local":     true,
	"other":     true,
}

// Validate checks all per-model YAML files under providersDir for structural
// correctness and returns any validation errors found.
func Validate(providersDir string) ([]ValidationError, error) {
	pattern := filepath.Join(providersDir, "*", "models", "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", pattern, err)
	}

	var allErrors []ValidationError

	for _, path := range matches {
		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		entry, err := ReadModelYAML(data)
		if err != nil {
			allErrors = append(allErrors, ValidationError{
				File:    path,
				Field:   "yaml",
				Message: fmt.Sprintf("failed to parse: %v", err),
			})
			continue
		}

		fileErrors := validateEntry(entry, path, providersDir)
		allErrors = append(allErrors, fileErrors...)
	}

	return allErrors, nil
}

func validateEntry(entry Entry, filePath, providersDir string) []ValidationError {
	var errs []ValidationError

	isWrapper := entry.Extends != ""

	// Required fields — provider and model_id are always required.
	if entry.Provider == "" {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "provider",
			Message: "required field is empty",
		})
	}
	if entry.ModelID == "" {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "model_id",
			Message: "required field is empty",
		})
	}

	// display_name and mode are required only for non-wrapper entries.
	if !isWrapper {
		if entry.DisplayName == "" {
			errs = append(errs, ValidationError{
				File:    filePath,
				Field:   "display_name",
				Message: "required field is empty",
			})
		}
		if entry.Mode == "" {
			errs = append(errs, ValidationError{
				File:    filePath,
				Field:   "mode",
				Message: "required field is empty",
			})
		}
	}

	// Provider match: extract provider directory name from path
	// Path pattern: providersDir/<provider>/models/<file>.yaml
	relPath, err := filepath.Rel(providersDir, filePath)
	if err == nil {
		parts := strings.SplitN(filepath.ToSlash(relPath), "/", 3)
		if len(parts) >= 1 && entry.Provider != "" && entry.Provider != parts[0] {
			errs = append(errs, ValidationError{
				File:    filePath,
				Field:   "provider",
				Message: fmt.Sprintf("value %q does not match directory %q", entry.Provider, parts[0]),
			})
		}
	}

	// Mode enum
	if entry.Mode != "" && !validModes[entry.Mode] {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "mode",
			Message: fmt.Sprintf("invalid value %q; must be one of: chat, embedding, image, audio_in, audio_out", entry.Mode),
		})
	}

	// Status enum
	if entry.Lifecycle.Status != "" && !validStatuses[entry.Lifecycle.Status] {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "lifecycle.status",
			Message: fmt.Sprintf("invalid value %q; must be one of: preview, ga, deprecated, sunset, legacy", entry.Lifecycle.Status),
		})
	}

	// Tier enum
	if entry.Tier != "" && !validTiers[entry.Tier] {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "tier",
			Message: fmt.Sprintf("invalid value %q; must be one of: flagship, standard", entry.Tier),
		})
	}

	// Agent-routing optional metadata enum validation. Missing values are
	// valid; only non-empty malformed values are rejected.
	errs = append(errs, validateAgentRouting(&entry, filePath)...)

	// Local/3rd-party benchmark source enum validation.
	if entry.Benchmarks != nil && entry.Benchmarks.Coding != nil && entry.Benchmarks.Coding.Source != "" && !validCodingBenchmarkSources[entry.Benchmarks.Coding.Source] {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "benchmarks.coding.source",
			Message: fmt.Sprintf("invalid value %q; must be one of: swe-bench, local, other", entry.Benchmarks.Coding.Source),
		})
	}

	return errs
}

func validateAgentRouting(entry *Entry, filePath string) []ValidationError {
	var errs []ValidationError
	ar := entry.AgentRouting
	if ar == nil {
		return errs
	}

	if ar.CodingQualityTier != "" && !validCodingTiers[ar.CodingQualityTier] {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "agent_routing.coding_quality_tier",
			Message: fmt.Sprintf("invalid value %q; must be one of: frontier, strong, balanced, fast, experimental, unknown", ar.CodingQualityTier),
		})
	}
	if ar.ReasoningQualityTier != "" && !validCodingTiers[ar.ReasoningQualityTier] {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "agent_routing.reasoning_quality_tier",
			Message: fmt.Sprintf("invalid value %q; must be one of: frontier, strong, balanced, fast, experimental, unknown", ar.ReasoningQualityTier),
		})
	}
	if ar.ToolUseQualityTier != "" && !validToolUseTiers[ar.ToolUseQualityTier] {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "agent_routing.tool_use_quality_tier",
			Message: fmt.Sprintf("invalid value %q; must be one of: strong, balanced, weak, unknown", ar.ToolUseQualityTier),
		})
	}
	if ar.LatencyTier != "" && !validLatencyTiers[ar.LatencyTier] {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "agent_routing.latency_tier",
			Message: fmt.Sprintf("invalid value %q; must be one of: low, medium, high, unknown", ar.LatencyTier),
		})
	}
	if ar.LocalSuitability != "" && !validLocalSuitability[ar.LocalSuitability] {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "agent_routing.local_suitability",
			Message: fmt.Sprintf("invalid value %q; must be one of: excellent, good, poor, unknown", ar.LocalSuitability),
		})
	}
	return errs
}

// CountProvidersAndModels returns the number of distinct providers and model
// YAML files under providersDir.
func CountProvidersAndModels(providersDir string) (int, int, error) {
	pattern := filepath.Join(providersDir, "*", "models", "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return 0, 0, fmt.Errorf("glob %s: %w", pattern, err)
	}

	providers := make(map[string]bool)
	for _, path := range matches {
		relPath, err := filepath.Rel(providersDir, path)
		if err != nil {
			continue
		}
		parts := strings.SplitN(filepath.ToSlash(relPath), "/", 3)
		if len(parts) >= 1 {
			providers[parts[0]] = true
		}
	}

	return len(providers), len(matches), nil
}

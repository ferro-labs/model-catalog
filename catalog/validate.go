package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
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

var requiredPricingFields = []string{
	"input_per_m_tokens",
	"output_per_m_tokens",
	"cache_read_per_m_tokens",
	"cache_write_per_m_tokens",
	"reasoning_per_m_tokens",
	"image_per_tile",
	"audio_input_per_minute",
	"audio_output_per_character",
	"embedding_per_m_tokens",
	"finetune_train_per_m_tokens",
	"finetune_input_per_m_tokens",
	"finetune_output_per_m_tokens",
}

var requiredCapabilityFields = []string{
	"vision",
	"audio_input",
	"audio_output",
	"function_calling",
	"parallel_tool_calls",
	"json_mode",
	"response_schema",
	"prompt_caching",
	"reasoning",
	"streaming",
	"finetuneable",
}

type yamlPresence struct {
	topLevel map[string]bool
	nested   map[string]map[string]bool
}

func inspectYAMLFields(data []byte) (yamlPresence, error) {
	presence := yamlPresence{
		topLevel: make(map[string]bool),
		nested:   make(map[string]map[string]bool),
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return presence, err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return presence, fmt.Errorf("model YAML must be a mapping")
	}

	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		val := root.Content[i+1]
		presence.topLevel[key] = true
		if val.Kind != yaml.MappingNode {
			continue
		}
		children := make(map[string]bool)
		for j := 0; j+1 < len(val.Content); j += 2 {
			children[val.Content[j].Value] = true
		}
		presence.nested[key] = children
	}

	return presence, nil
}

func (p yamlPresence) hasTopLevel(field string) bool {
	return p.topLevel[field]
}

func (p yamlPresence) hasNested(parent, field string) bool {
	children := p.nested[parent]
	return children != nil && children[field]
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

		presence, err := inspectYAMLFields(data)
		if err != nil {
			allErrors = append(allErrors, ValidationError{
				File:    path,
				Field:   "yaml",
				Message: fmt.Sprintf("failed to inspect fields: %v", err),
			})
			continue
		}

		fileErrors := validateEntry(entry, path, providersDir, presence)
		allErrors = append(allErrors, fileErrors...)
	}

	return allErrors, nil
}

func validateEntry(entry Entry, filePath, providersDir string, presence yamlPresence) []ValidationError {
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

	if entry.Lifecycle.Status == "" {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "lifecycle.status",
			Message: "required field is empty",
		})
	}

	if entry.Tier == "" {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "tier",
			Message: "required field is empty",
		})
	}

	if !presence.hasTopLevel("pricing") {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "pricing",
			Message: "required block is missing",
		})
	} else {
		for _, field := range requiredPricingFields {
			if !presence.hasNested("pricing", field) {
				errs = append(errs, ValidationError{
					File:    filePath,
					Field:   "pricing." + field,
					Message: "required field is missing",
				})
			}
		}
	}

	if !presence.hasTopLevel("capabilities") {
		errs = append(errs, ValidationError{
			File:    filePath,
			Field:   "capabilities",
			Message: "required block is missing",
		})
	} else {
		for _, field := range requiredCapabilityFields {
			if !presence.hasNested("capabilities", field) {
				errs = append(errs, ValidationError{
					File:    filePath,
					Field:   "capabilities." + field,
					Message: "required field is missing",
				})
			}
		}
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

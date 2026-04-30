package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// MigrateExtends migrates wrapper provider models to use extends: base/*
// wrappers. It reads models from both providers, identifies common model_ids,
// and rewrites wrapper files with minimal YAML that extends the base.
func MigrateExtends(providersDir, wrapperProvider, baseProvider string, dryRun bool) error {
	baseDir := filepath.Join(providersDir, baseProvider, "models")
	wrapperDir := filepath.Join(providersDir, wrapperProvider, "models")

	// Read all base provider models into a map keyed by model_id.
	baseEntries, err := ReadProviderModels(baseDir)
	if err != nil {
		return fmt.Errorf("read %s models: %w", baseProvider, err)
	}

	// Read all wrapper provider models into a map keyed by model_id.
	wrapperEntries, err := ReadProviderModels(wrapperDir)
	if err != nil {
		return fmt.Errorf("read %s models: %w", wrapperProvider, err)
	}

	migrated := 0
	for modelID, wrapperEntry := range wrapperEntries {
		// Skip models that already have an extends reference.
		if wrapperEntry.Extends != "" {
			continue
		}

		baseEntry, ok := baseEntries[modelID]
		if !ok {
			continue
		}

		if dryRun {
			fmt.Printf("[dry-run] Would migrate %s/%s → extends: %s/%s\n",
				wrapperProvider, modelID, baseProvider, modelID)
			migrated++
			continue
		}

		// Compute the minimal wrapper YAML.
		wrapperYAML := ComputeWrapperYAML(baseEntry, wrapperEntry, modelID, baseProvider)

		// Write the wrapper YAML to the wrapper provider file.
		// Use the same filename convention: replace / with __ for model_ids with slashes.
		filename := strings.ReplaceAll(modelID, "/", "__") + ".yaml"
		outPath := filepath.Join(wrapperDir, filename)
		if err := os.WriteFile(outPath, wrapperYAML, 0o644); err != nil {
			return fmt.Errorf("write wrapper %s: %w", outPath, err)
		}

		migrated++
	}

	if dryRun {
		fmt.Printf("[dry-run] Would migrate %d %s models to extends: %s/* wrappers\n",
			migrated, wrapperProvider, baseProvider)
	} else {
		fmt.Printf("Migrated %d %s models to extends: %s/* wrappers\n",
			migrated, wrapperProvider, baseProvider)
	}
	return nil
}

// ReadProviderModels reads all YAML files in a models directory and returns
// entries keyed by model_id.
func ReadProviderModels(modelsDir string) (map[string]Entry, error) {
	pattern := filepath.Join(modelsDir, "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", pattern, err)
	}

	entries := make(map[string]Entry, len(matches))
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		entry, err := ReadModelYAML(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}

		entries[entry.ModelID] = entry
	}

	return entries, nil
}

// ComputeWrapperYAML computes the minimal YAML for a wrapper entry.
// It includes only the fields that differ from the base entry, plus the
// mandatory extends/provider/model_id/display_name/tier fields.
func ComputeWrapperYAML(base, wrapper Entry, modelID, baseProvider string) []byte {
	// Build a minimal wrapper struct for YAML serialization.
	// We use an ordered map approach to control field order.
	var doc yaml.Node
	doc.Kind = yaml.DocumentNode

	mapping := &yaml.Node{Kind: yaml.MappingNode}
	doc.Content = append(doc.Content, mapping)

	// Always include these fields.
	AddStringField(mapping, "extends", baseProvider+"/"+modelID)
	AddStringField(mapping, "provider", wrapper.Provider)
	AddStringField(mapping, "model_id", modelID)
	AddStringField(mapping, "display_name", wrapper.DisplayName)

	// Mode: never include (cannot be overridden per ResolveExtends).
	// But if it differs, that's an error - the extends resolver will reject it.

	// context_window: include only if different.
	if wrapper.ContextWindow != base.ContextWindow && wrapper.ContextWindow != 0 {
		AddIntField(mapping, "context_window", wrapper.ContextWindow)
	}

	// max_output_tokens: include only if different.
	if wrapper.MaxOutputTokens != base.MaxOutputTokens && wrapper.MaxOutputTokens != 0 {
		AddIntField(mapping, "max_output_tokens", wrapper.MaxOutputTokens)
	}

	// Pricing: ALWAYS include all 12 fields. The extends resolver does a full
	// replacement of pricing (it cannot distinguish "not set" from "explicitly
	// null"), so the wrapper must specify every field.
	pricingNode := PricingToYAML(wrapper.Pricing)
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "pricing"},
		pricingNode,
	)

	// Capabilities: ALWAYS include all fields (bare bools, can't distinguish
	// "not set" from "false").
	capsNode := CapabilitiesToYAML(wrapper.Capabilities)
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "capabilities"},
		capsNode,
	)

	// Lifecycle: include only fields that differ.
	lifecycleNode := computeLifecycleDiff(base.Lifecycle, wrapper.Lifecycle)
	if lifecycleNode != nil {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "lifecycle"},
			lifecycleNode,
		)
	}

	// source: include only if different.
	if wrapper.Source != base.Source && wrapper.Source != "" {
		AddStringField(mapping, "source", wrapper.Source)
	}

	// updated_at: include only if different.
	if wrapper.UpdatedAt != base.UpdatedAt && wrapper.UpdatedAt != "" {
		AddStringField(mapping, "updated_at", wrapper.UpdatedAt)
	}

	// tier: always include.
	if wrapper.Tier != "" {
		AddStringField(mapping, "tier", wrapper.Tier)
	}

	out, _ := yaml.Marshal(&doc)
	return out
}

// computeLifecycleDiff returns a YAML mapping node with only the lifecycle fields
// that differ between base and wrapper. Returns nil if no differences.
func computeLifecycleDiff(base, wrapper Lifecycle) *yaml.Node {
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	hasDiff := false

	if wrapper.Status != base.Status && wrapper.Status != "" {
		hasDiff = true
		AddStringField(mapping, "status", wrapper.Status)
	}

	if ptrStringDiffers(base.DeprecationDate, wrapper.DeprecationDate) {
		hasDiff = true
		AddPtrStringField(mapping, "deprecation_date", wrapper.DeprecationDate)
	}

	if ptrStringDiffers(base.SunsetDate, wrapper.SunsetDate) {
		hasDiff = true
		AddPtrStringField(mapping, "sunset_date", wrapper.SunsetDate)
	}

	if ptrStringDiffers(base.Successor, wrapper.Successor) {
		hasDiff = true
		AddPtrStringField(mapping, "successor", wrapper.Successor)
	}

	if !hasDiff {
		return nil
	}
	return mapping
}

// ptrStringDiffers returns true if two *string pointers have different values.
func ptrStringDiffers(a, b *string) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil || b == nil {
		return true
	}
	return *a != *b
}

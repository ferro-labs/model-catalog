package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var versionSuffixRe = regexp.MustCompile(`-v\d+:\d+$`)
var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]`)

var providerToBedrockVendor = map[string]string{
	"anthropic":  "anthropic",
	"meta_llama": "meta",
	"mistral":    "mistral",
	"cohere":     "cohere",
	"ai21":       "ai21",
}

// extractCoreName extracts the core model name from a Bedrock-style model ID
// by stripping the vendor prefix and version suffix.
// "us.anthropic.claude-3-haiku-20240307-v1:0" with vendor "anthropic"
// returns "claude-3-haiku-20240307".
func extractCoreName(modelID, vendor string) string {
	needle := vendor + "."
	idx := strings.Index(modelID, needle)
	if idx < 0 {
		return ""
	}
	core := modelID[idx+len(needle):]
	core = versionSuffixRe.ReplaceAllString(core, "")
	return core
}

// normalizeForFuzzy strips all non-alphanumeric chars and lowercases for
// fuzzy comparison between Bedrock-style IDs (meta.llama3-3-70b-instruct)
// and upstream IDs (Llama-3.3-70B-Instruct).
func normalizeForFuzzy(s string) string {
	return nonAlphanumRe.ReplaceAllString(strings.ToLower(s), "")
}

// fuzzyNormalizeModelName normalizes a model name for comparison by
// lowercasing, stripping non-alphanumeric chars, and removing known
// hardware/quantization suffixes (fp8, fp16, 128e, 16e, etc.).
func fuzzyNormalizeModelName(s string) string {
	n := nonAlphanumRe.ReplaceAllString(strings.ToLower(s), "")
	for _, suffix := range []string{"fp8", "fp16"} {
		n = strings.TrimSuffix(n, suffix)
	}
	// Remove expert-count segments like "128e", "16e" before "instruct"
	if idx := strings.Index(n, "instruct"); idx > 0 {
		prefix := n[:idx]
		for _, pat := range []string{"128e", "16e", "8e"} {
			prefix = strings.Replace(prefix, pat, "", 1)
		}
		n = prefix + n[idx:]
	}
	return n
}

// findBaseMatch tries to match a wrapper model ID against base entries.
// Exact match first, then normalized matching for Bedrock-style IDs,
// then fuzzy matching (stripped alphanumeric comparison) for providers
// like meta_llama where naming conventions diverge significantly.
func findBaseMatch(wrapperModelID string, baseEntries map[string]Entry, baseProvider string) (string, Entry, bool) {
	if entry, ok := baseEntries[wrapperModelID]; ok {
		return wrapperModelID, entry, true
	}

	vendor, ok := providerToBedrockVendor[baseProvider]
	if !ok {
		return "", Entry{}, false
	}

	coreName := extractCoreName(wrapperModelID, vendor)
	if coreName == "" {
		return "", Entry{}, false
	}

	if entry, ok := baseEntries[coreName]; ok {
		return coreName, entry, true
	}

	normalizedCore := fuzzyNormalizeModelName(coreName)
	for baseID, entry := range baseEntries {
		normalizedBase := fuzzyNormalizeModelName(baseID)
		if normalizedCore == normalizedBase {
			return baseID, entry, true
		}
	}

	return "", Entry{}, false
}

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
	skipped := 0
	for modelID, wrapperEntry := range wrapperEntries {
		if wrapperEntry.Extends != "" {
			continue
		}

		baseModelID, baseEntry, ok := findBaseMatch(modelID, baseEntries, baseProvider)
		if !ok {
			skipped++
			continue
		}

		if dryRun {
			fmt.Printf("[dry-run] Would migrate %s/%s → extends: %s/%s\n",
				wrapperProvider, modelID, baseProvider, baseModelID)
			migrated++
			continue
		}

		wrapperYAML := ComputeWrapperYAML(baseEntry, wrapperEntry, baseModelID, baseProvider)

		filename := SanitizeFilename(modelID) + ".yaml"
		outPath := filepath.Join(wrapperDir, filename)
		if err := os.WriteFile(outPath, wrapperYAML, 0o600); err != nil {
			return fmt.Errorf("write wrapper %s: %w", outPath, err)
		}

		migrated++
	}

	if dryRun {
		fmt.Printf("[dry-run] Would migrate %d %s models to extends: %s/* wrappers (%d skipped — no base match)\n",
			migrated, wrapperProvider, baseProvider, skipped)
	} else {
		fmt.Printf("Migrated %d %s models to extends: %s/* wrappers (%d skipped — no base match)\n",
			migrated, wrapperProvider, baseProvider, skipped)
	}
	return nil
}

// BackfillExtendsLifecycle adds full lifecycle blocks to extends wrappers
// that are missing them. It resolves the correct lifecycle by merging
// base and wrapper values. This is needed because the extends resolver does
// full replacement of lifecycle (same as pricing/capabilities).
func BackfillExtendsLifecycle(providersDir string) (int, error) {
	allEntries := make(map[string]Entry)
	provDirs, err := filepath.Glob(filepath.Join(providersDir, "*/models"))
	if err != nil {
		return 0, fmt.Errorf("glob providers: %w", err)
	}
	for _, modelsDir := range provDirs {
		entries, err := ReadProviderModels(modelsDir)
		if err != nil {
			return 0, fmt.Errorf("read %s: %w", modelsDir, err)
		}
		provider := filepath.Base(filepath.Dir(modelsDir))
		for modelID, entry := range entries {
			allEntries[provider+"/"+modelID] = entry
		}
	}

	fixed := 0
	for key, entry := range allEntries {
		if entry.Extends == "" {
			continue
		}

		base, ok := allEntries[entry.Extends]
		if !ok {
			continue
		}

		// Compute resolved lifecycle: base values as defaults, wrapper overrides.
		resolved := base.Lifecycle
		if entry.Lifecycle.Status != "" {
			resolved.Status = entry.Lifecycle.Status
		}
		if entry.Lifecycle.DeprecationDate != nil {
			resolved.DeprecationDate = entry.Lifecycle.DeprecationDate
		}
		if entry.Lifecycle.SunsetDate != nil {
			resolved.SunsetDate = entry.Lifecycle.SunsetDate
		}
		if entry.Lifecycle.Successor != nil {
			resolved.Successor = entry.Lifecycle.Successor
		}

		provider := strings.SplitN(key, "/", 2)[0]
		filename := SanitizeFilename(entry.ModelID) + ".yaml"
		path := filepath.Join(providersDir, provider, "models", filename)

		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return 0, fmt.Errorf("read %s: %w", path, err)
		}

		content := string(data)
		if strings.Contains(content, "\nlifecycle:") || strings.Contains(content, "\nlifecycle:\n") {
			continue
		}

		lcBlock := fmt.Sprintf("lifecycle:\n    status: %s\n    deprecation_date: %s\n    sunset_date: %s\n    successor: %s\n",
			resolved.Status,
			ptrStringToYAML(resolved.DeprecationDate),
			ptrStringToYAML(resolved.SunsetDate),
			ptrStringToYAML(resolved.Successor),
		)

		lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
		var result []string
		inserted := false
		for _, line := range lines {
			if !inserted && (strings.HasPrefix(line, "source:") || strings.HasPrefix(line, "updated_at:") || strings.HasPrefix(line, "tier:")) {
				result = append(result, strings.TrimRight(lcBlock, "\n"))
				inserted = true
			}
			result = append(result, line)
		}
		if !inserted {
			result = append(result, strings.TrimRight(lcBlock, "\n"))
		}

		if err := os.WriteFile(filepath.Clean(path), []byte(strings.Join(result, "\n")+"\n"), 0o600); err != nil { //nolint:gosec // path from filepath.Glob
			return 0, fmt.Errorf("write %s: %w", path, err)
		}
		fixed++
	}

	return fixed, nil
}

const yamlNull = "null"

func ptrStringToYAML(s *string) string {
	if s == nil {
		return yamlNull
	}
	return *s
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
		data, err := os.ReadFile(filepath.Clean(path))
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
func ComputeWrapperYAML(base, wrapper Entry, baseModelID, baseProvider string) []byte {
	// Build a minimal wrapper struct for YAML serialization.
	// We use an ordered map approach to control field order.
	var doc yaml.Node
	doc.Kind = yaml.DocumentNode

	mapping := &yaml.Node{Kind: yaml.MappingNode}
	doc.Content = append(doc.Content, mapping)

	AddStringField(mapping, "extends", baseProvider+"/"+baseModelID)
	AddStringField(mapping, "provider", wrapper.Provider)
	AddStringField(mapping, "model_id", wrapper.ModelID)
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

	// Lifecycle: ALWAYS include all fields. Like pricing/capabilities, YAML
	// deserialization can't distinguish "not set" from "explicitly null" for
	// *string fields, so the wrapper must specify every lifecycle field.
	lifecycleNode := LifecycleToYAML(wrapper.Lifecycle)
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "lifecycle"},
		lifecycleNode,
	)

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


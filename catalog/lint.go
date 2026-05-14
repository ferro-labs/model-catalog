package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// LintIssue represents a single lint finding.
type LintIssue struct {
	Severity string // "error" or "warning"
	File     string
	Key      string // catalog key (provider/model_id)
	Message  string
}

// dimensionPattern matches segments like "1024-x-1024", "512-x-512", "max-x-max".
var dimensionPattern = regexp.MustCompile(`(?:^|/)(\d+-x-\d+|max-x-max)(?:/|$)`)

// parameterSegment matches standalone "steps", "width", "height", or "quality"
// as path segments (between "/" delimiters).
var parameterSegment = regexp.MustCompile(`(?:^|/)(?:steps|\d+-steps|max-steps|width|height|quality)(?:/|$)`)

// IsJunkKey returns true if the catalog key contains dimension patterns or
// parameter segments indicating it is a parameterized request, not a real model.
func IsJunkKey(key string) bool {
	return dimensionPattern.MatchString(key) || parameterSegment.MatchString(key)
}

// Lint detects junk keys and duplicate model IDs across providers in the
// YAML files under providersDir.
func Lint(providersDir string) ([]LintIssue, error) {
	pattern := filepath.Join(providersDir, "*", "models", "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", pattern, err)
	}

	var issues []LintIssue

	// Track model_id -> list of (provider, file) for duplicate detection.
	type modelRef struct {
		provider string
		file     string
		key      string
		extends  string // non-empty if this entry uses extends
	}
	modelIndex := make(map[string][]modelRef)

	for _, path := range matches {
		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		entry, err := ReadModelYAML(data)
		if err != nil {
			// Skip unparseable files; validate catches those.
			continue
		}

		key := entry.Provider + "/" + entry.ModelID

		// Check 1: junk keys.
		if IsJunkKey(key) {
			issues = append(issues, LintIssue{
				Severity: "error",
				File:     path,
				Key:      key,
				Message:  "junk key contains dimension or parameter segments",
			})
		}

		// Check 2: ga models without a source URL (extends wrappers inherit from base).
		if entry.Extends == "" && entry.Lifecycle.Status == "ga" && entry.Source == "" {
			issues = append(issues, LintIssue{
				Severity: "warning",
				File:     path,
				Key:      key,
				Message:  "ga model has empty source field",
			})
		}

		// Collect for duplicate detection.
		modelIndex[entry.ModelID] = append(modelIndex[entry.ModelID], modelRef{
			provider: entry.Provider,
			file:     path,
			key:      key,
			extends:  entry.Extends,
		})
	}

	// Check 3: duplicate model IDs across providers.
	// Skip groups where any entry uses extends (intentional inheritance).
	var dupModelIDs []string
	for modelID, refs := range modelIndex {
		providers := make(map[string]bool)
		hasExtends := false
		for _, r := range refs {
			providers[r.provider] = true
			if r.extends != "" {
				hasExtends = true
			}
		}
		if len(providers) <= 1 || hasExtends {
			continue
		}
		dupModelIDs = append(dupModelIDs, modelID)
	}
	sort.Strings(dupModelIDs)

	for _, modelID := range dupModelIDs {
		refs := modelIndex[modelID]
		var providerNames []string
		seen := make(map[string]bool)
		for _, r := range refs {
			if !seen[r.provider] {
				providerNames = append(providerNames, r.provider)
				seen[r.provider] = true
			}
		}
		sort.Strings(providerNames)

		msg := fmt.Sprintf("duplicate model_id %q found in providers: %s (candidate for extends)",
			modelID, strings.Join(providerNames, ", "))

		for _, r := range refs {
			issues = append(issues, LintIssue{
				Severity: "warning",
				File:     r.file,
				Key:      r.key,
				Message:  msg,
			})
		}
	}

	return issues, nil
}

// CountModels returns the number of model YAML files under providersDir.
func CountModels(providersDir string) (int, error) {
	pattern := filepath.Join(providersDir, "*", "models", "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return 0, fmt.Errorf("glob %s: %w", pattern, err)
	}
	return len(matches), nil
}

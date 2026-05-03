package catalog

import "fmt"

// ResolveExtends takes all raw entries (including wrappers with Extends set)
// and returns resolved entries with inheritance applied.
// Rules:
//   - Max chain depth = 1 (wrapper cannot extend another wrapper)
//   - Deep merge: wrapper scalars win, maps merge recursively
//   - Mode cannot be overridden
//   - Provider and ModelID are always required in the wrapper
func ResolveExtends(entries map[string]Entry) (map[string]Entry, error) {
	resolved := make(map[string]Entry, len(entries))

	for key, entry := range entries {
		if entry.Extends == "" {
			resolved[key] = entry
			continue
		}

		// Wrapper must have provider and model_id set.
		if entry.Provider == "" || entry.ModelID == "" {
			return nil, fmt.Errorf("entry %q extends %q but is missing provider or model_id", key, entry.Extends)
		}

		// Look up the base entry.
		base, ok := entries[entry.Extends]
		if !ok {
			return nil, fmt.Errorf("entry %q extends %q which does not exist", key, entry.Extends)
		}

		// Max chain depth = 1: base must not itself extend anything.
		if base.Extends != "" {
			return nil, fmt.Errorf("entry %q extends %q which itself extends %q: max chain depth is 1", key, entry.Extends, base.Extends)
		}

		merged, err := mergeEntries(base, entry)
		if err != nil {
			return nil, fmt.Errorf("merging %q with base %q: %w", key, entry.Extends, err)
		}

		// Clear the extends field so it never appears in output.
		merged.Extends = ""
		resolved[key] = merged
	}

	return resolved, nil
}

// mergeEntries merges a wrapper entry on top of a base entry.
// The wrapper's non-zero/non-empty values win over the base.
func mergeEntries(base, wrapper Entry) (Entry, error) {
	result := base

	// Mode cannot be overridden: if wrapper sets a different mode, error.
	if wrapper.Mode != "" && wrapper.Mode != base.Mode {
		return Entry{}, fmt.Errorf("mode cannot be overridden: base=%q, wrapper=%q", base.Mode, wrapper.Mode)
	}

	// String fields: wrapper wins if non-empty.
	if wrapper.Provider != "" {
		result.Provider = wrapper.Provider
	}
	if wrapper.ModelID != "" {
		result.ModelID = wrapper.ModelID
	}
	if wrapper.DisplayName != "" {
		result.DisplayName = wrapper.DisplayName
	}
	if wrapper.Source != "" {
		result.Source = wrapper.Source
	}
	if wrapper.UpdatedAt != "" {
		result.UpdatedAt = wrapper.UpdatedAt
	}
	if wrapper.Tier != "" {
		result.Tier = wrapper.Tier
	}
	// Mode is kept from base (already validated it can't change).

	// Int fields: wrapper wins if non-zero.
	if wrapper.ContextWindow != 0 {
		result.ContextWindow = wrapper.ContextWindow
	}
	if wrapper.MaxOutputTokens != 0 {
		result.MaxOutputTokens = wrapper.MaxOutputTokens
	}

	// Pricing: full replacement from wrapper (same rationale as Capabilities —
	// YAML deserialization can't distinguish "not set" from "explicitly null",
	// so wrapper must specify all pricing fields).
	result.Pricing = wrapper.Pricing

	// Capabilities: wrapper always overrides all fields (bare bool, can't
	// distinguish "explicitly false" from "not set").
	result.Capabilities = wrapper.Capabilities

	// Lifecycle: full replacement from wrapper. Like pricing/capabilities,
	// YAML deserialization can't distinguish "not set" from "explicitly null"
	// for *string fields, so wrapper must specify all lifecycle fields.
	result.Lifecycle = wrapper.Lifecycle

	return result, nil
}

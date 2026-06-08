package scrape

import "regexp"

var (
	anthropicFamilyVersionRE     = regexp.MustCompile(`^claude-(opus|sonnet|haiku)-([0-9]+)\.([0-9]+)($|-.*)$`)
	anthropicGenerationVersionRE = regexp.MustCompile(`^claude-([0-9]+)\.([0-9]+)-(.+)$`)
)

// NormalizeObservations returns a copy of observations with provider/model IDs
// converted to the catalog canonical identity form before grouping.
func NormalizeObservations(observations []Observation) []Observation {
	normalized := make([]Observation, len(observations))
	for i, obs := range observations {
		normalized[i] = NormalizeObservation(obs)
	}
	return normalized
}

// NormalizeObservation returns a copy of obs with provider-specific model ID
// aliases converted to the catalog canonical identity form.
func NormalizeObservation(obs Observation) Observation {
	obs.Provider, obs.ModelID = NormalizeProviderModel(obs.Provider, obs.ModelID)
	return obs
}

// NormalizeProviderModel canonicalizes model identity across scrape sources.
func NormalizeProviderModel(provider, modelID string) (string, string) {
	switch provider {
	case "anthropic":
		modelID = normalizeAnthropicModelID(modelID)
	}
	return provider, modelID
}

func normalizeAnthropicModelID(modelID string) string {
	modelID = anthropicFamilyVersionRE.ReplaceAllString(modelID, "claude-$1-$2-$3$4")
	modelID = anthropicGenerationVersionRE.ReplaceAllString(modelID, "claude-$1-$2-$3")
	return modelID
}

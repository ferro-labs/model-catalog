package scrape

import (
	"github.com/ferro-labs/model-catalog/catalog"
)

// ContextProof records that one or more scraped sources confirmed a catalog
// entry's context fields (context_window / max_output_tokens), and how confident
// that confirmation is.
type ContextProof struct {
	Key        string // provider/model_id
	Provider   string
	ModelID    string
	Confidence Confidence
	Sources    []Observation // observations whose context matched the catalog
}

// VerifyContext returns, for each catalog entry, the observations whose context
// fields agree with the catalog. Confidence is high when >=2 sources agree and
// medium for a single source — mirroring VerifyPricing. Entries with no agreeing
// source are omitted. Output is sorted by key.
func VerifyContext(entries map[string]catalog.Entry, observations []Observation) []ContextProof {
	observations = DedupObservations(NormalizeObservations(observations))

	byKey := make(map[string][]Observation)
	for _, o := range observations {
		byKey[o.Provider+"/"+o.ModelID] = append(byKey[o.Provider+"/"+o.ModelID], o)
	}

	var proofs []ContextProof
	for _, key := range sortedKeys(byKey) {
		entry, ok := entries[key]
		if !ok {
			continue
		}
		var agree []Observation
		for _, o := range byKey[key] {
			if contextAgrees(entry, o) {
				agree = append(agree, o)
			}
		}
		if len(agree) == 0 {
			continue
		}
		conf := ConfidenceMedium
		if len(agree) >= 2 {
			conf = ConfidenceHigh
		}
		proofs = append(proofs, ContextProof{
			Key:        key,
			Provider:   entry.Provider,
			ModelID:    entry.ModelID,
			Confidence: conf,
			Sources:    agree,
		})
	}
	return proofs
}

// contextAgrees reports whether every context field the observation carries
// matches the catalog entry exactly. Fields the observation does not carry are
// ignored. At least one field must be present and match.
func contextAgrees(e catalog.Entry, o Observation) bool {
	checked := 0
	match := func(cat int, obs *int) bool {
		if obs == nil {
			return true // not observed → no disagreement
		}
		checked++
		return cat == *obs
	}
	if !match(e.ContextWindow, o.ContextWindow) {
		return false
	}
	if !match(e.MaxOutputTokens, o.MaxOutput) {
		return false
	}
	return checked > 0
}

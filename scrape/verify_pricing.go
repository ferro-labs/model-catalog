package scrape

import (
	"math"

	"github.com/ferro-labs/model-catalog/catalog"
)

// PricingProof records that one or more scraped sources confirmed a catalog
// entry's pricing, and how confident that confirmation is.
type PricingProof struct {
	Key        string // provider/model_id
	Provider   string
	ModelID    string
	Confidence Confidence
	Sources    []Observation // observations whose pricing matched the catalog
}

// VerifyPricing returns, for each catalog entry, the observations whose pricing
// agrees with the catalog within tolerance. Confidence is high when >=2 sources
// agree and medium for a single source — the same thresholds Reconcile uses.
// Entries with no agreeing source are omitted. Output is sorted by key.
func VerifyPricing(entries map[string]catalog.Entry, observations []Observation) []PricingProof {
	observations = DedupObservations(NormalizeObservations(observations))

	byKey := make(map[string][]Observation)
	for _, o := range observations {
		byKey[o.Provider+"/"+o.ModelID] = append(byKey[o.Provider+"/"+o.ModelID], o)
	}

	var proofs []PricingProof
	for _, key := range sortedKeys(byKey) {
		entry, ok := entries[key]
		if !ok {
			continue
		}
		var agree []Observation
		for _, o := range byKey[key] {
			if pricingAgrees(entry.Pricing, o) {
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
		proofs = append(proofs, PricingProof{
			Key:        key,
			Provider:   entry.Provider,
			ModelID:    entry.ModelID,
			Confidence: conf,
			Sources:    agree,
		})
	}
	return proofs
}

// pricingAgrees reports whether every price field the observation carries
// matches the catalog entry within floatTolerance. Fields the observation does
// not carry are ignored; a catalog N/A against an observed price is a
// disagreement. At least one field must be present and match.
func pricingAgrees(p catalog.Pricing, o Observation) bool {
	checked := 0
	match := func(cat catalog.NullFloat64, obs *float64) bool {
		if obs == nil {
			return true // not observed → no disagreement
		}
		checked++
		if !cat.Valid {
			return false // catalog says N/A but source reports a price
		}
		return math.Abs(cat.Value-*obs) <= floatTolerance
	}
	if !match(p.InputPerMTokens, o.InputPerM) {
		return false
	}
	if !match(p.OutputPerMTokens, o.OutputPerM) {
		return false
	}
	if !match(p.CacheReadPerMTokens, o.CacheReadPerM) {
		return false
	}
	return checked > 0
}

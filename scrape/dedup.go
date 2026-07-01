package scrape

import (
	"sort"
	"strconv"
)

// DedupObservations guarantees at most one observation per
// (provider, model_id, source): a single source must never contribute two
// values for the same catalog key, which would make it "disagree with itself"
// and corrupt the confidence grouping in Reconcile.
//
// Duplicates arise when distinct upstream keys collapse to the same catalog
// identity — e.g. LiteLLM lists both "deepseek-chat" and "deepseek/deepseek-chat",
// or normalization maps two aliases onto one model. Apply this AFTER
// NormalizeObservations so it also catches normalization-induced collisions.
//
// The survivor is chosen deterministically (sorted by identity then price
// signature, keep first), so the result does not depend on scraper/map iteration
// order — otherwise the weekly review file could churn and open spurious PRs.
func DedupObservations(obs []Observation) []Observation {
	sorted := make([]Observation, len(obs))
	copy(sorted, obs)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Provider != b.Provider {
			return a.Provider < b.Provider
		}
		if a.ModelID != b.ModelID {
			return a.ModelID < b.ModelID
		}
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return priceSignature(a) < priceSignature(b)
	})

	out := make([]Observation, 0, len(sorted))
	seen := make(map[string]struct{}, len(sorted))
	for _, o := range sorted {
		key := o.Provider + "\x00" + o.ModelID + "\x00" + o.Source
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, o)
	}
	return out
}

// priceSignature is a stable string form of an observation's price fields, used
// only as a deterministic tie-breaker when the same (provider, model, source)
// appears more than once with differing values.
func priceSignature(o Observation) string {
	return fmtPtr(o.InputPerM) + "|" + fmtPtr(o.OutputPerM) + "|" + fmtPtr(o.CacheReadPerM)
}

func fmtPtr(p *float64) string {
	if p == nil {
		return "nil"
	}
	return strconv.FormatFloat(*p, 'f', -1, 64)
}

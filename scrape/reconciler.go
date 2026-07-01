package scrape

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/ferro-labs/model-catalog/catalog"
)

const floatTolerance = 0.001

// Delta represents a single field difference between our catalog and scraped data.
type Delta struct {
	CatalogKey string
	Field      string
	Current    string
	Scraped    string
	// ScrapedValue is the raw agreed-upon scraped number for single-value diffs
	// (high/medium confidence). It is unset for conflict diffs, where sources
	// disagree and Scraped holds a composite string instead.
	ScrapedValue float64
	Confidence   Confidence
	Sources      []string
}

// ReconcileResult holds the outcome of cross-checking scraped observations against the catalog.
type ReconcileResult struct {
	Checked   int
	Matches   int
	Diffs     []Delta
	NewModels []string
	Missing   []string
}

// Reconcile cross-checks scraped observations against the catalog and returns a result
// summarizing matches, differences, new models, and missing models.
func Reconcile(entries map[string]catalog.Entry, observations []Observation) ReconcileResult {
	// Normalize then dedup so a single source can never contribute two values
	// for one catalog key (which would make it disagree with itself and produce
	// order-dependent conflict output). Both are idempotent, so callers that
	// already applied them lose nothing.
	observations = NormalizeObservations(observations)
	observations = DedupObservations(observations)

	// Group observations by catalog key (provider/model_id).
	grouped := make(map[string][]Observation)
	for _, obs := range observations {
		key := obs.Provider + "/" + obs.ModelID
		grouped[key] = append(grouped[key], obs)
	}

	// Track which catalog entries have at least one observation.
	catalogSeen := make(map[string]bool)

	var result ReconcileResult
	var newModels []string

	// Sorted keys for deterministic output.
	scrapedKeys := sortedKeys(grouped)

	for _, key := range scrapedKeys {
		obs := grouped[key]
		entry, exists := entries[key]
		if !exists {
			newModels = append(newModels, key)
			continue
		}

		catalogSeen[key] = true
		result.Checked++

		diffs := compareEntry(key, entry, obs)
		if len(diffs) == 0 {
			result.Matches++
		} else {
			result.Diffs = append(result.Diffs, diffs...)
		}
	}

	result.NewModels = newModels

	// Find catalog entries not seen in any scraper.
	catalogKeys := sortedKeys(entries)
	for _, key := range catalogKeys {
		if !catalogSeen[key] {
			result.Missing = append(result.Missing, key)
		}
	}

	return result
}

// compareEntry checks pricing fields of a catalog entry against observations.
func compareEntry(key string, entry catalog.Entry, obs []Observation) []Delta {
	type fieldVal struct {
		source string
		value  *float64
	}

	type fieldGroup struct {
		catalogFieldName string
		catalogValue     float64
		catalogValid     bool
		observations     []fieldVal
	}

	fields := []fieldGroup{
		{
			catalogFieldName: "pricing.input_per_m_tokens",
			catalogValue:     entry.Pricing.InputPerMTokens.Value,
			catalogValid:     entry.Pricing.InputPerMTokens.Valid,
		},
		{
			catalogFieldName: "pricing.output_per_m_tokens",
			catalogValue:     entry.Pricing.OutputPerMTokens.Value,
			catalogValid:     entry.Pricing.OutputPerMTokens.Valid,
		},
		{
			catalogFieldName: "pricing.cache_read_per_m_tokens",
			catalogValue:     entry.Pricing.CacheReadPerMTokens.Value,
			catalogValid:     entry.Pricing.CacheReadPerMTokens.Valid,
		},
	}

	// Gather observed values per field.
	for _, o := range obs {
		fields[0].observations = append(fields[0].observations, fieldVal{source: o.Source, value: o.InputPerM})
		fields[1].observations = append(fields[1].observations, fieldVal{source: o.Source, value: o.OutputPerM})
		fields[2].observations = append(fields[2].observations, fieldVal{source: o.Source, value: o.CacheReadPerM})
	}

	var diffs []Delta

	for _, f := range fields {
		// Collect non-nil observed values.
		var observed []sourceValue
		for _, fv := range f.observations {
			if fv.value != nil {
				observed = append(observed, sourceValue{source: fv.source, value: *fv.value})
			}
		}

		if len(observed) == 0 {
			continue
		}

		// Group observed values to detect agreement/conflict.
		groups := groupByValue(observed)

		if len(groups) == 1 {
			// All sources agree.
			scrapedVal := observed[0].value
			var sources []string
			for _, sv := range observed {
				sources = appendUnique(sources, sv.source)
			}
			sort.Strings(sources) // deterministic display regardless of scrape order

			if !f.catalogValid {
				// Catalog has null, scrapers have a value -> diff.
				confidence := ConfidenceMedium
				if len(sources) >= 2 {
					confidence = ConfidenceHigh
				}
				diffs = append(diffs, Delta{
					CatalogKey:   key,
					Field:        f.catalogFieldName,
					Current:      "null",
					Scraped:      formatFloat(scrapedVal),
					ScrapedValue: scrapedVal,
					Confidence:   confidence,
					Sources:      sources,
				})
				continue
			}

			if !floatsEqual(f.catalogValue, scrapedVal) {
				confidence := ConfidenceMedium
				if len(sources) >= 2 {
					confidence = ConfidenceHigh
				}
				diffs = append(diffs, Delta{
					CatalogKey:   key,
					Field:        f.catalogFieldName,
					Current:      formatFloat(f.catalogValue),
					Scraped:      formatFloat(scrapedVal),
					ScrapedValue: scrapedVal,
					Confidence:   confidence,
					Sources:      sources,
				})
			}
		} else {
			// Sources disagree.
			var allSources []string
			var scrapedVals []string
			for _, g := range groups {
				for _, sv := range g {
					allSources = appendUnique(allSources, sv.source)
				}
				scrapedVals = append(scrapedVals, fmt.Sprintf("%s=%s",
					g[0].source, formatFloat(g[0].value)))
			}
			// Deterministic display regardless of scrape/grouping order.
			sort.Strings(allSources)
			sort.Strings(scrapedVals)

			currentStr := "null"
			if f.catalogValid {
				currentStr = formatFloat(f.catalogValue)
			}

			diffs = append(diffs, Delta{
				CatalogKey: key,
				Field:      f.catalogFieldName,
				Current:    currentStr,
				Scraped:    strings.Join(scrapedVals, ", "),
				Confidence: ConfidenceConflict,
				Sources:    allSources,
			})
		}
	}

	return diffs
}

// floatsEqual returns true if a and b are within floatTolerance.
func floatsEqual(a, b float64) bool {
	return math.Abs(a-b) <= floatTolerance
}

func formatFloat(v float64) string {
	return fmt.Sprintf("%.4f", v)
}

type sourceValue struct {
	source string
	value  float64
}

// groupByValue groups sourceValues by their float value (within tolerance).
func groupByValue(svs []sourceValue) [][]sourceValue {
	var groups [][]sourceValue

	for _, sv := range svs {
		placed := false
		for i := range groups {
			if floatsEqual(groups[i][0].value, sv.value) {
				groups[i] = append(groups[i], sv)
				placed = true
				break
			}
		}
		if !placed {
			groups = append(groups, []sourceValue{sv})
		}
	}

	return groups
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

// sortedKeys returns the sorted keys of a map (generic over value type).
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

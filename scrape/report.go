package scrape

import (
	"fmt"
	"sort"
	"strings"
)

// FormatWeeklyReview renders a deterministic, timestamp-free markdown document
// of the price diffs that are NOT auto-applied — single-source ("medium") and
// conflicting diffs — for human review in the weekly PR. Corroborated (≥2
// sources) diffs are excluded because they are applied directly to the YAML.
//
// Output is stable across runs (diffs are sorted), so an unchanged week produces
// an identical file and therefore no pull request.
func FormatWeeklyReview(result ReconcileResult) string {
	med := filterDiffs(result.Diffs, ConfidenceMedium)
	conflict := filterDiffs(result.Diffs, ConfidenceConflict)
	sortDiffs(med)
	sortDiffs(conflict)

	var b strings.Builder
	b.WriteString("# Weekly Price Review\n\n")
	b.WriteString("Corroborated changes (≥2 sources agree) are auto-applied in this PR's file diffs.\n")
	b.WriteString("The diffs below come from a single source or disagree between sources — review and\n")
	b.WriteString("apply them manually to the model YAML if correct.\n\n")

	fmt.Fprintf(&b, "## Single-source price diffs (%d)\n\n", len(med))
	if len(med) == 0 {
		b.WriteString("_None._\n\n")
	} else {
		for _, d := range med {
			fmt.Fprintf(&b, "- `%s` `%s`: %s → %s (%s only)\n",
				d.CatalogKey, d.Field, d.Current, d.Scraped, strings.Join(d.Sources, " + "))
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "## Conflicting price diffs (%d)\n\n", len(conflict))
	if len(conflict) == 0 {
		b.WriteString("_None._\n")
	} else {
		for _, d := range conflict {
			fmt.Fprintf(&b, "- `%s` `%s`: catalog=%s, sources=%s\n",
				d.CatalogKey, d.Field, d.Current, d.Scraped)
		}
	}

	return b.String()
}

// sortDiffs orders diffs deterministically by catalog key then field.
func sortDiffs(diffs []Delta) {
	sort.Slice(diffs, func(i, j int) bool {
		if diffs[i].CatalogKey != diffs[j].CatalogKey {
			return diffs[i].CatalogKey < diffs[j].CatalogKey
		}
		return diffs[i].Field < diffs[j].Field
	})
}

// FormatReport produces a human-readable summary of the reconciliation result.
func FormatReport(result ReconcileResult) string {
	var b strings.Builder

	b.WriteString("Scrape Report\n")
	b.WriteString(strings.Repeat("─", 40) + "\n")
	fmt.Fprintf(&b, "Models checked:      %d\n", result.Checked)
	fmt.Fprintf(&b, "Matches:             %d\n", result.Matches)
	fmt.Fprintf(&b, "Price differences:   %d\n", len(result.Diffs))
	fmt.Fprintf(&b, "New models found:    %d\n", len(result.NewModels))
	fmt.Fprintf(&b, "Missing from scrape: %d\n", len(result.Missing))
	b.WriteString("\n")

	// Group diffs by confidence.
	highDiffs := filterDiffs(result.Diffs, ConfidenceHigh)
	medDiffs := filterDiffs(result.Diffs, ConfidenceMedium)
	conflictDiffs := filterDiffs(result.Diffs, ConfidenceConflict)

	if len(highDiffs) > 0 {
		fmt.Fprintf(&b, "PRICE DIFFERENCES (high confidence): %d\n", len(highDiffs))
		for _, d := range highDiffs {
			fmt.Fprintf(&b, "  %s [%s]: %s → %s (%s)\n",
				d.CatalogKey, d.Field, d.Current, d.Scraped, strings.Join(d.Sources, " + "))
		}
		b.WriteString("\n")
	}

	if len(medDiffs) > 0 {
		fmt.Fprintf(&b, "PRICE DIFFERENCES (medium confidence): %d\n", len(medDiffs))
		for _, d := range medDiffs {
			fmt.Fprintf(&b, "  %s [%s]: %s → %s (%s only)\n",
				d.CatalogKey, d.Field, d.Current, d.Scraped, strings.Join(d.Sources, " + "))
		}
		b.WriteString("\n")
	}

	if len(conflictDiffs) > 0 {
		fmt.Fprintf(&b, "PRICE DIFFERENCES (conflict): %d\n", len(conflictDiffs))
		for _, d := range conflictDiffs {
			fmt.Fprintf(&b, "  %s [%s]: catalog=%s, scraped=%s (%s)\n",
				d.CatalogKey, d.Field, d.Current, d.Scraped, strings.Join(d.Sources, " + "))
		}
		b.WriteString("\n")
	}

	if len(result.NewModels) > 0 {
		fmt.Fprintf(&b, "NEW MODELS (not in catalog): %d\n", len(result.NewModels))
		limit := len(result.NewModels)
		if limit > 50 {
			limit = 50
		}
		for _, m := range result.NewModels[:limit] {
			fmt.Fprintf(&b, "  %s\n", m)
		}
		if len(result.NewModels) > 50 {
			fmt.Fprintf(&b, "  ... and %d more\n", len(result.NewModels)-50)
		}
		b.WriteString("\n")
	}

	if len(result.Missing) > 0 {
		fmt.Fprintf(&b, "MISSING FROM SCRAPE (in catalog but not scraped): %d\n", len(result.Missing))
		limit := len(result.Missing)
		if limit > 30 {
			limit = 30
		}
		for _, m := range result.Missing[:limit] {
			fmt.Fprintf(&b, "  %s\n", m)
		}
		if len(result.Missing) > 30 {
			fmt.Fprintf(&b, "  ... and %d more\n", len(result.Missing)-30)
		}
	}

	return b.String()
}

func filterDiffs(diffs []Delta, conf Confidence) []Delta {
	var out []Delta
	for _, d := range diffs {
		if d.Confidence == conf {
			out = append(out, d)
		}
	}
	return out
}

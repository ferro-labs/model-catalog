package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/ferro-labs/model-catalog/scrape"
	"github.com/ferro-labs/model-catalog/scrape/oracle"
	"github.com/spf13/cobra"
)

var (
	scrapeReportFile  string
	scrapeAutoAdd     bool
	scrapeWrite       bool
	scrapeApplyPrices bool
)

func init() {
	scrapeCmd.Flags().StringVar(&scrapeReportFile, "report", "", "write report to file (e.g., report.md)")
	scrapeCmd.Flags().BoolVar(&scrapeAutoAdd, "auto-add", false, "generate YAML for new models found in scrapers")
	scrapeCmd.Flags().BoolVar(&scrapeWrite, "write", false, "write auto-added and applied files (default is dry-run)")
	scrapeCmd.Flags().BoolVar(&scrapeApplyPrices, "apply-prices", false, "apply high-confidence price diffs to existing model YAML instead of failing")
	rootCmd.AddCommand(scrapeCmd)
}

var scrapeCmd = &cobra.Command{
	Use:   "scrape",
	Short: "Scrape external APIs and cross-check against the catalog",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScrape()
	},
}

func runScrape() error {
	// Load catalog from dist/catalog.json.
	catalogPath := "dist/catalog.json"
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return fmt.Errorf("read catalog: %w (run 'ferrocat build' first)", err)
	}

	var entries map[string]catalog.Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parse catalog: %w", err)
	}

	fmt.Printf("Loaded %d catalog entries from %s\n", len(entries), catalogPath)

	// Run all scrapers.
	scrapers := []scrape.Scraper{
		oracle.NewOpenRouter(),
		oracle.NewModelsDev(),
	}

	var allObs []scrape.Observation
	var failedScrapers []string
	for _, s := range scrapers {
		fmt.Printf("Scraping %s...\n", s.Name())
		obs, err := s.Scrape()
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: %s scraper failed: %v\n", s.Name(), err)
			failedScrapers = append(failedScrapers, s.Name())
			continue
		}
		fmt.Printf("  %s: %d models fetched\n", s.Name(), len(obs))
		allObs = append(allObs, obs...)
	}

	if len(allObs) == 0 {
		return fmt.Errorf("no observations collected from any scraper")
	}

	allObs = scrape.NormalizeObservations(allObs)

	// Reconcile.
	result := scrape.Reconcile(entries, allObs)
	report := scrape.FormatReport(result)

	fmt.Println()
	fmt.Print(report)

	// Optionally write to file.
	if scrapeReportFile != "" {
		if err := os.WriteFile(scrapeReportFile, []byte(report), 0o600); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
		fmt.Printf("\nReport written to %s\n", scrapeReportFile)
	}

	// Auto-add new models if requested.
	if scrapeAutoAdd && len(result.NewModels) > 0 {
		candidates := buildAutoAddCandidates(result.NewModels, allObs)
		dryRun := !scrapeWrite

		addResult, err := catalog.AutoAdd("providers", candidates, dryRun)
		if err != nil {
			return fmt.Errorf("auto-add: %w", err)
		}

		fmt.Printf("\nAuto-add: %d added, %d skipped (exists), %d skipped (no provider folder)\n",
			addResult.Added, addResult.Skipped, addResult.NoProvider)
	}

	// Apply high-confidence price diffs to existing models if requested. These
	// land in the weekly PR for human review rather than failing the run.
	if scrapeApplyPrices {
		var updates []catalog.PriceUpdate
		for _, d := range result.Diffs {
			if d.Confidence != scrape.ConfidenceHigh {
				continue
			}
			parts := strings.SplitN(d.CatalogKey, "/", 2)
			if len(parts) != 2 {
				continue
			}
			updates = append(updates, catalog.PriceUpdate{
				Provider: parts[0],
				ModelID:  parts[1],
				Field:    strings.TrimPrefix(d.Field, "pricing."),
				Value:    d.ScrapedValue,
			})
		}
		if len(updates) > 0 {
			now := time.Now().UTC().Format("2006-01-02")
			applyResult, err := catalog.ApplyPriceUpdates("providers", updates, now, !scrapeWrite)
			if err != nil {
				return fmt.Errorf("apply prices: %w", err)
			}
			fmt.Printf("\nApply prices: %d field(s) across %d file(s) applied, %d could not be applied\n",
				applyResult.Applied, applyResult.Files, len(applyResult.NotApplied))
			for _, u := range applyResult.NotApplied {
				fmt.Printf("  NOT APPLIED: %s\n", u)
			}
		}
	}

	// Exit code: when not auto-applying, fail on high-confidence diffs so the
	// command still works as a standalone freshness check. With --apply-prices,
	// corroborated diffs are written to the PR and conflicts/un-appliable diffs
	// are report-only, so the run stays green.
	if !scrapeApplyPrices {
		for _, d := range result.Diffs {
			if d.Confidence == scrape.ConfidenceHigh {
				return fmt.Errorf("actionable high-confidence diffs found")
			}
		}
	}

	if len(failedScrapers) > 0 {
		return fmt.Errorf("scrapers failed: %s", strings.Join(failedScrapers, ", "))
	}

	return nil
}

// buildAutoAddCandidates filters new models to only those with high confidence
// (≥2 oracles agree on pricing).
func buildAutoAddCandidates(newModels []string, allObs []scrape.Observation) []catalog.AutoAddCandidate {
	allObs = scrape.NormalizeObservations(allObs)

	grouped := make(map[string][]scrape.Observation)
	for _, obs := range allObs {
		key := obs.Provider + "/" + obs.ModelID
		grouped[key] = append(grouped[key], obs)
	}

	var candidates []catalog.AutoAddCandidate
	for _, key := range newModels {
		obs := grouped[key]
		if len(obs) < 2 {
			continue
		}

		parts := strings.SplitN(key, "/", 2)
		if len(parts) != 2 {
			continue
		}

		inputPrices := make(map[string]*float64)
		outputPrices := make(map[string]*float64)
		var sources []string

		for _, o := range obs {
			inputPrices[o.Source] = o.InputPerM
			outputPrices[o.Source] = o.OutputPerM
			sources = appendUniqueStr(sources, o.Source)
		}

		inputAgreed, inputVal := pricesAgree(inputPrices)
		outputAgreed, outputVal := pricesAgree(outputPrices)

		if !inputAgreed && !outputAgreed {
			continue
		}

		sort.Strings(sources)
		candidates = append(candidates, catalog.AutoAddCandidate{
			Provider:   parts[0],
			ModelID:    parts[1],
			InputPerM:  inputVal,
			OutputPerM: outputVal,
			Sources:    sources,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Provider+"/"+candidates[i].ModelID < candidates[j].Provider+"/"+candidates[j].ModelID
	})
	return candidates
}

func pricesAgree(prices map[string]*float64) (bool, *float64) {
	var vals []*float64
	for _, v := range prices {
		if v != nil {
			vals = append(vals, v)
		}
	}
	if len(vals) < 2 {
		return false, nil
	}
	first := *vals[0]
	for _, v := range vals[1:] {
		if *v != first {
			return false, nil
		}
	}
	return true, vals[0]
}

func appendUniqueStr(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/ferro-labs/model-catalog/scrape"
	"github.com/ferro-labs/model-catalog/scrape/oracle"
	"github.com/spf13/cobra"
)

var scrapeReportFile string

func init() {
	scrapeCmd.Flags().StringVar(&scrapeReportFile, "report", "", "write report to file (e.g., report.md)")
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
	for _, s := range scrapers {
		fmt.Printf("Scraping %s...\n", s.Name())
		obs, err := s.Scrape()
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: %s scraper failed: %v\n", s.Name(), err)
			continue
		}
		fmt.Printf("  %s: %d models fetched\n", s.Name(), len(obs))
		allObs = append(allObs, obs...)
	}

	if len(allObs) == 0 {
		return fmt.Errorf("no observations collected from any scraper")
	}

	// Reconcile.
	result := scrape.Reconcile(entries, allObs)
	report := scrape.FormatReport(result)

	fmt.Println()
	fmt.Print(report)

	// Optionally write to file.
	if scrapeReportFile != "" {
		if err := os.WriteFile(scrapeReportFile, []byte(report), 0o644); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
		fmt.Printf("\nReport written to %s\n", scrapeReportFile)
	}

	// Exit code: 1 if there are high-confidence diffs.
	for _, d := range result.Diffs {
		if d.Confidence == scrape.ConfidenceHigh {
			fmt.Println("\nActionable high-confidence diffs found. Exiting with code 1.")
			os.Exit(1)
		}
	}

	return nil
}

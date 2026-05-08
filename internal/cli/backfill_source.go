package cli

import (
	"fmt"
	"os"

	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/ferro-labs/model-catalog/scrape"
	"github.com/ferro-labs/model-catalog/scrape/oracle"
	"github.com/spf13/cobra"
)

var backfillDryRun bool

func init() {
	backfillSourceCmd.Flags().BoolVar(&backfillDryRun, "dry-run", false, "Print changes without writing")
	rootCmd.AddCommand(backfillSourceCmd)
}

var backfillSourceCmd = &cobra.Command{
	Use:   "backfill-source",
	Short: "Fill empty source fields from oracle data",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBackfillSource()
	},
}

func runBackfillSource() error {
	// Phase 1: Static provider-level URLs (no network calls).
	fmt.Println("Phase 1: Backfilling from provider URL map...")
	staticUpdated, err := catalog.BackfillSourceFromProviderURLs("providers", backfillDryRun)
	if err != nil {
		return fmt.Errorf("static backfill: %w", err)
	}
	if backfillDryRun {
		fmt.Printf("[dry-run] Would update %d models from provider URLs\n\n", staticUpdated)
	} else {
		fmt.Printf("Updated %d models from provider URLs\n\n", staticUpdated)
	}

	// Phase 2: Scraper-based URLs for remaining gaps.
	fmt.Println("Phase 2: Backfilling from oracle scrapers...")
	scrapers := []scrape.Scraper{
		oracle.NewOpenRouter(),
		oracle.NewModelsDev(),
	}

	sourceMap := make(map[string]string)
	for _, s := range scrapers {
		fmt.Printf("Scraping %s...\n", s.Name())
		obs, err := s.Scrape()
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: %s failed: %v\n", s.Name(), err)
			continue
		}
		fmt.Printf("  %s: %d models\n", s.Name(), len(obs))

		for _, o := range obs {
			key := o.Provider + "/" + o.ModelID
			if _, exists := sourceMap[key]; !exists {
				sourceMap[key] = buildSourceURL(o)
			}
		}
	}

	if len(sourceMap) == 0 {
		fmt.Println("No oracle observations collected, skipping phase 2")
		return nil
	}

	fmt.Printf("Collected source URLs for %d models\n\n", len(sourceMap))

	scraperUpdated, err := catalog.BackfillSource("providers", sourceMap, backfillDryRun)
	if err != nil {
		return err
	}

	total := staticUpdated + scraperUpdated
	if backfillDryRun {
		fmt.Printf("\n[dry-run] Would update %d models total (%d provider URLs + %d oracle)\n",
			total, staticUpdated, scraperUpdated)
	} else {
		fmt.Printf("\nUpdated %d models total (%d provider URLs + %d oracle)\n",
			total, staticUpdated, scraperUpdated)
	}
	return nil
}

func buildSourceURL(obs scrape.Observation) string {
	switch obs.Source {
	case "openrouter":
		return "https://openrouter.ai/models/" + obs.Provider + "/" + obs.ModelID
	case "models_dev":
		return "https://models.dev/" + obs.Provider + "/" + obs.ModelID
	default:
		return obs.SourceURL
	}
}

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
		return fmt.Errorf("no observations collected from any scraper")
	}

	fmt.Printf("Collected source URLs for %d models\n\n", len(sourceMap))

	updated, err := catalog.BackfillSource("providers", sourceMap, backfillDryRun)
	if err != nil {
		return err
	}

	if backfillDryRun {
		fmt.Printf("\n[dry-run] Would update %d models\n", updated)
	} else {
		fmt.Printf("\nUpdated %d models with source URLs\n", updated)
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

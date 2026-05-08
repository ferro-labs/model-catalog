package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/ferro-labs/model-catalog/scrape"
	"github.com/ferro-labs/model-catalog/scrape/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(freshnessCmd)
}

var freshnessCmd = &cobra.Command{
	Use:   "freshness",
	Short: "Check catalog freshness against provider APIs",
	Long:  "Queries provider /v1/models endpoints and flags models that exist upstream but not in the catalog.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFreshness()
	},
}

func runFreshness() error {
	catalogPath := "dist/catalog.json"
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return fmt.Errorf("read catalog: %w (run 'ferrocat build' first)", err)
	}

	var entries map[string]catalog.Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parse catalog: %w", err)
	}

	type providerCheck struct {
		name    string
		scraper scrape.Scraper
	}

	var checks []providerCheck

	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		checks = append(checks, providerCheck{"anthropic", api.NewAnthropic(key)})
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		checks = append(checks, providerCheck{"openai", api.NewOpenAI(key)})
	}

	if len(checks) == 0 {
		fmt.Println("No API keys configured. Set env vars to enable provider checks:")
		fmt.Println("  ANTHROPIC_API_KEY  — check anthropic models")
		fmt.Println("  OPENAI_API_KEY     — check openai models")
		return nil
	}

	totalMissing := 0

	for _, check := range checks {
		fmt.Printf("Checking %s...\n", check.name)
		obs, err := check.scraper.Scrape()
		if err != nil {
			fmt.Fprintf(os.Stderr, "  WARNING: %s failed: %v\n", check.name, err)
			continue
		}

		catalogModels := make(map[string]bool)
		for key := range entries {
			if strings.HasPrefix(key, check.name+"/") {
				modelID := strings.TrimPrefix(key, check.name+"/")
				catalogModels[modelID] = true
			}
		}

		var missing []string
		for _, o := range obs {
			if !catalogModels[o.ModelID] {
				missing = append(missing, o.ModelID)
			}
		}
		sort.Strings(missing)

		fmt.Printf("  API: %d models, Catalog: %d models\n", len(obs), len(catalogModels))

		if len(missing) == 0 {
			fmt.Printf("  ✓ No missing models\n")
		} else {
			fmt.Printf("  ✗ %d models in API but not in catalog:\n", len(missing))
			for _, m := range missing {
				fmt.Printf("    - %s/%s\n", check.name, m)
			}
			totalMissing += len(missing)
		}
		fmt.Println()
	}

	if totalMissing > 0 {
		fmt.Printf("FRESHNESS CHECK FAILED: %d models missing from catalog\n", totalMissing)
		os.Exit(1)
	}

	fmt.Println("FRESHNESS CHECK PASSED: catalog is up to date with provider APIs")
	return nil
}

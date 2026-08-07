package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/ferro-labs/model-catalog/scrape"
	"github.com/ferro-labs/model-catalog/scrape/oracle"
	"github.com/spf13/cobra"
)

var (
	snapshotDir    string
	snapshotDryRun bool
)

func init() {
	snapshotProvenanceCmd.Flags().StringVar(&snapshotDir, "snapshot-dir", "snapshots", "directory for content-addressed price snapshots (pushed to the snapshots branch)")
	snapshotProvenanceCmd.Flags().BoolVar(&snapshotDryRun, "dry-run", false, "report upgrades without writing snapshots or YAML")
	rootCmd.AddCommand(snapshotProvenanceCmd)
}

var snapshotProvenanceCmd = &cobra.Command{
	Use:   "snapshot-provenance",
	Short: "Upgrade sources.pricing to snapshot-verified provenance from public oracles",
	Long: "Runs the public pricing oracles (openrouter, models.dev, litellm) and, for every\n" +
		"catalog model whose pricing an oracle confirms, archives a content-addressed price\n" +
		"snapshot under --snapshot-dir and upgrades that model's sources.pricing to verified\n" +
		"provenance (verified_by: scraper-<oracle>, confidence high/medium, snapshot_sha256 +\n" +
		"snapshot_branch). Never downgrades a higher-confidence group.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSnapshotProvenance()
	},
}

func runSnapshotProvenance() error {
	catalogPath := "dist/catalog.json"
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return fmt.Errorf("read catalog: %w (run 'ferrocat build' first)", err)
	}
	var entries map[string]catalog.Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parse catalog: %w", err)
	}

	scrapers := []scrape.Scraper{
		oracle.NewOpenRouter(),
		oracle.NewModelsDev(),
		oracle.NewLiteLLM(),
	}
	var allObs []scrape.Observation
	for _, s := range scrapers {
		fmt.Printf("Scraping %s...\n", s.Name())
		obs, scrapeErr := s.Scrape()
		if scrapeErr != nil {
			fmt.Fprintf(os.Stderr, "  WARNING: %s failed: %v\n", s.Name(), scrapeErr)
			continue
		}
		fmt.Printf("  %s: %d models\n", s.Name(), len(obs))
		allObs = append(allObs, obs...)
	}
	if len(allObs) == 0 {
		return fmt.Errorf("no observations collected from any oracle")
	}

	proofs := scrape.VerifyPricing(entries, allObs)
	fmt.Printf("\n%d catalog models have oracle-confirmed pricing\n", len(proofs))

	today := time.Now().UTC().Format("2006-01-02")
	store := catalog.SnapshotStore{Dir: snapshotDir}

	var ups []catalog.ProvenanceUpgrade
	for _, p := range proofs {
		src := p.Sources[0] // primary agreeing oracle
		verifiedBy := "scraper-" + src.Source
		snap := catalog.PriceSnapshot{
			Provider:      p.Provider,
			ModelID:       p.ModelID,
			SourceURL:     src.SourceURL,
			VerifiedBy:    verifiedBy,
			VerifiedAt:    today,
			InputPerM:     src.InputPerM,
			OutputPerM:    src.OutputPerM,
			CacheReadPerM: src.CacheReadPerM,
		}
		content, marshalErr := snap.CanonicalJSON()
		if marshalErr != nil {
			return fmt.Errorf("%s: %w", p.Key, marshalErr)
		}

		var sha string
		if snapshotDryRun {
			sha = catalog.SnapshotSHA256(content)
		} else if sha, err = store.Put(content); err != nil {
			return err
		}

		shaCopy, branchCopy := sha, catalog.DefaultSnapshotBranch
		ups = append(ups, catalog.ProvenanceUpgrade{
			Provider: p.Provider,
			ModelID:  p.ModelID,
			Group:    "pricing",
			Prov: catalog.Provenance{
				URL:            src.SourceURL,
				VerifiedAt:     today,
				Confidence:     string(p.Confidence),
				VerifiedBy:     verifiedBy,
				SnapshotSHA256: &shaCopy,
				SnapshotBranch: &branchCopy,
			},
		})
	}

	if snapshotDryRun {
		fmt.Printf("[dry-run] would upgrade sources.pricing on %d models and write %d snapshot(s) to %s/\n",
			len(ups), len(ups), snapshotDir)
		return nil
	}

	applied, skipped, err := catalog.ApplyProvenanceUpgrades("providers", ups)
	if err != nil {
		return err
	}
	fmt.Printf("Upgraded %d models (%d skipped: monotonicity or non-trailing sources block)\n", applied, skipped)
	fmt.Printf("Snapshots written to %s/ — push to the %q branch with 'make snapshots-push'\n",
		snapshotDir, catalog.DefaultSnapshotBranch)
	return nil
}

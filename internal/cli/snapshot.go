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

	priceProofs := scrape.VerifyPricing(entries, allObs)
	contextProofs := scrape.VerifyContext(entries, allObs)
	fmt.Printf("\noracle-confirmed: %d models pricing, %d models context\n", len(priceProofs), len(contextProofs))

	today := time.Now().UTC().Format("2006-01-02")
	store := catalog.SnapshotStore{Dir: snapshotDir}

	var ups []catalog.ProvenanceUpgrade
	// addUpgrade archives one group's snapshot and appends a provenance upgrade.
	// Shared by the pricing and context passes so the store/provenance
	// boilerplate lives in one place. src is the primary agreeing observation:
	// its SourceURL and Source name feed the recorded provenance.
	addUpgrade := func(provider, modelID, group string, src scrape.Observation, conf scrape.Confidence, snapshot []byte) error {
		var sha string
		if snapshotDryRun {
			sha = catalog.SnapshotSHA256(snapshot)
		} else if s, putErr := store.Put(snapshot); putErr != nil {
			return putErr
		} else {
			sha = s
		}
		shaCopy, branchCopy := sha, catalog.DefaultSnapshotBranch
		ups = append(ups, catalog.ProvenanceUpgrade{
			Provider: provider,
			ModelID:  modelID,
			Group:    group,
			Prov: catalog.Provenance{
				URL:            src.SourceURL,
				VerifiedAt:     today,
				Confidence:     string(conf),
				VerifiedBy:     "scraper-" + src.Source,
				SnapshotSHA256: &shaCopy,
				SnapshotBranch: &branchCopy,
			},
		})
		return nil
	}

	for _, p := range priceProofs {
		src := p.Sources[0] // primary agreeing oracle
		content, marshalErr := catalog.PriceSnapshot{
			Provider: p.Provider, ModelID: p.ModelID, SourceURL: src.SourceURL,
			InputPerM: src.InputPerM, OutputPerM: src.OutputPerM, CacheReadPerM: src.CacheReadPerM,
		}.CanonicalJSON()
		if marshalErr != nil {
			return fmt.Errorf("%s pricing: %w", p.Key, marshalErr)
		}
		if err := addUpgrade(p.Provider, p.ModelID, "pricing", src, p.Confidence, content); err != nil {
			return err
		}
	}
	for _, p := range contextProofs {
		src := p.Sources[0]
		content, marshalErr := catalog.ContextSnapshot{
			Provider: p.Provider, ModelID: p.ModelID, SourceURL: src.SourceURL,
			ContextWindow: src.ContextWindow, MaxOutputTokens: src.MaxOutput,
		}.CanonicalJSON()
		if marshalErr != nil {
			return fmt.Errorf("%s context: %w", p.Key, marshalErr)
		}
		if err := addUpgrade(p.Provider, p.ModelID, "context", src, p.Confidence, content); err != nil {
			return err
		}
	}

	if snapshotDryRun {
		fmt.Printf("[dry-run] would apply %d provenance upgrade(s) and write their snapshot(s) to %s/\n",
			len(ups), snapshotDir)
		return nil
	}

	applied, skipped, err := catalog.ApplyProvenanceUpgrades("providers", ups)
	if err != nil {
		return err
	}
	fmt.Printf("Applied %d provenance upgrade(s) (%d skipped: monotonicity/idempotent/non-trailing sources)\n", applied, skipped)
	fmt.Printf("Snapshots written to %s/ — push to the %q branch with 'make snapshots-push'\n",
		snapshotDir, catalog.DefaultSnapshotBranch)
	return nil
}

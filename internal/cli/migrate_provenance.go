package cli

import (
	"fmt"

	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/spf13/cobra"
)

var migrateProvenanceDryRun bool

func init() {
	migrateProvenanceCmd.Flags().BoolVar(&migrateProvenanceDryRun, "dry-run", false, "Print counts without writing")
	rootCmd.AddCommand(migrateProvenanceCmd)
}

var migrateProvenanceCmd = &cobra.Command{
	Use:   "migrate-provenance",
	Short: "Seed baseline sources.pricing provenance from legacy source/updated_at",
	RunE: func(cmd *cobra.Command, args []string) error {
		migrated, skipped, err := catalog.MigrateProvenance("providers", migrateProvenanceDryRun)
		if err != nil {
			return err
		}
		verb := "Migrated"
		if migrateProvenanceDryRun {
			verb = "[dry-run] Would migrate"
		}
		fmt.Printf("%s %d models (%d skipped: no source or non-YYYY-MM-DD date)\n", verb, migrated, skipped)
		return nil
	},
}

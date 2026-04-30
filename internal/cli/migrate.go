package cli

import (
	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/spf13/cobra"
)

var (
	flagWrapper string
	flagBase    string
	flagDryRun  bool
)

func init() {
	migrateExtendsCmd.Flags().StringVar(&flagWrapper, "wrapper", "", "Wrapper provider name (e.g., azure)")
	migrateExtendsCmd.Flags().StringVar(&flagBase, "base", "", "Base provider name (e.g., openai)")
	migrateExtendsCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Print what would be migrated without writing")
	_ = migrateExtendsCmd.MarkFlagRequired("wrapper")
	_ = migrateExtendsCmd.MarkFlagRequired("base")
	rootCmd.AddCommand(migrateExtendsCmd)
}

var migrateExtendsCmd = &cobra.Command{
	Use:   "migrate-extends",
	Short: "Migrate wrapper provider models to use extends: base/* wrappers",
	RunE: func(cmd *cobra.Command, args []string) error {
		return catalog.MigrateExtends("providers", flagWrapper, flagBase, flagDryRun)
	},
}

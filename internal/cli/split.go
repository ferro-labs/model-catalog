package cli

import (
	"fmt"
	"os"

	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/spf13/cobra"
)

var splitOutputDir string

func init() {
	splitCmd.Flags().StringVarP(&splitOutputDir, "output", "o", "providers", "output directory for provider/model YAML files")
	rootCmd.AddCommand(splitCmd)
}

var splitCmd = &cobra.Command{
	Use:   "split <catalog.json>",
	Short: "Split catalog JSON into per-model YAML files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("read catalog file: %w", err)
		}
		return catalog.Split(data, splitOutputDir)
	},
}

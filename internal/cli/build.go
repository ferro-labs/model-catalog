package cli

import (
	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/spf13/cobra"
)

var buildOutputDir string

func init() {
	buildCmd.Flags().StringVarP(&buildOutputDir, "output", "o", "dist", "output directory for catalog.json")
	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build catalog JSON from per-model YAML files",
	RunE: func(cmd *cobra.Command, args []string) error {
		return catalog.Build("providers", buildOutputDir)
	},
}

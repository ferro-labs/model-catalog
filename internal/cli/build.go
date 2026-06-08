package cli

import (
	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/spf13/cobra"
)

var buildOutputDir string
var buildVersion string

func init() {
	buildCmd.Flags().StringVarP(&buildOutputDir, "output", "o", "dist", "output directory for catalog.json")
	buildCmd.Flags().StringVar(&buildVersion, "version", "", "manifest version to write, e.g. v2026.06.08")
	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build catalog JSON from per-model YAML files",
	RunE: func(cmd *cobra.Command, args []string) error {
		return catalog.BuildWithVersion("providers", buildOutputDir, buildVersion)
	},
}

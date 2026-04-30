package cli

import (
	"fmt"
	"os"

	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/spf13/cobra"
)

var lintProvidersDir string

func init() {
	lintCmd.Flags().StringVarP(&lintProvidersDir, "providers", "p", "providers", "path to providers directory")
	rootCmd.AddCommand(lintCmd)
}

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Detect junk keys and duplicate model IDs in the catalog",
	RunE: func(cmd *cobra.Command, args []string) error {
		issues, err := catalog.Lint(lintProvidersDir)
		if err != nil {
			return err
		}

		// Partition into errors and warnings.
		var errors, warnings []catalog.LintIssue
		for _, issue := range issues {
			switch issue.Severity {
			case "error":
				errors = append(errors, issue)
			case "warning":
				warnings = append(warnings, issue)
			}
		}

		// Print errors first, then warnings.
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "ERROR: %s: %s\n", e.Key, e.Message)
		}
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "WARN: %s: %s\n", w.Key, w.Message)
		}

		// Count distinct models from files.
		modelCount, countErr := catalog.CountModels(lintProvidersDir)
		if countErr != nil {
			return countErr
		}

		fmt.Printf("Lint: %d errors, %d warnings across %d models\n", len(errors), len(warnings), modelCount)

		if len(errors) > 0 {
			os.Exit(1)
		}
		return nil
	},
}

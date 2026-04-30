package cli

import (
	"fmt"
	"os"

	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/spf13/cobra"
)

var validateProvidersDir string

func init() {
	validateCmd.Flags().StringVarP(&validateProvidersDir, "providers", "p", "providers", "path to providers directory")
	rootCmd.AddCommand(validateCmd)
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate structural correctness of per-model YAML files",
	RunE: func(cmd *cobra.Command, args []string) error {
		errs, err := catalog.Validate(validateProvidersDir)
		if err != nil {
			return err
		}

		for _, ve := range errs {
			fmt.Fprintf(os.Stderr, "ERROR: %s: %s: %s\n", ve.File, ve.Field, ve.Message)
		}

		providerCount, modelCount, countErr := catalog.CountProvidersAndModels(validateProvidersDir)
		if countErr != nil {
			return countErr
		}

		fmt.Printf("Validated %d models across %d providers: %d errors\n", modelCount, providerCount, len(errs))

		if len(errs) > 0 {
			os.Exit(1)
		}
		return nil
	},
}

package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ferrocat",
	Short: "Ferro Model Catalog CLI",
}

func Execute() error {
	return rootCmd.Execute()
}

package cli

import (
	"fmt"
	"time"

	"github.com/ferro-labs/model-catalog/catalog"
	"github.com/spf13/cobra"
)

var (
	pruneDays   int
	pruneCutoff string
	pruneDryRun bool
)

func init() {
	pruneCmd.Flags().IntVar(&pruneDays, "days", 90, "Prune models sunset more than this many days ago")
	pruneCmd.Flags().StringVar(&pruneCutoff, "cutoff", "", "Explicit cutoff date in YYYY-MM-DD format")
	pruneCmd.Flags().BoolVar(&pruneDryRun, "dry-run", false, "Print pruned models without deleting files")
	rootCmd.AddCommand(pruneCmd)
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove models whose sunset date is older than the retention window",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPrune()
	},
}

func runPrune() error {
	cutoff, err := pruneCutoffDate(time.Now().UTC(), pruneDays, pruneCutoff)
	if err != nil {
		return err
	}

	pruned, err := catalog.PruneSunset("providers", cutoff, pruneDryRun)
	if err != nil {
		return err
	}
	if pruneDryRun {
		fmt.Printf("[dry-run] Would prune %d models with sunset_date before %s\n", pruned, cutoff.Format("2006-01-02"))
	} else {
		fmt.Printf("Pruned %d models with sunset_date before %s\n", pruned, cutoff.Format("2006-01-02"))
	}
	return nil
}

func pruneCutoffDate(now time.Time, days int, explicit string) (time.Time, error) {
	if explicit != "" {
		cutoff, err := time.Parse("2006-01-02", explicit)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse --cutoff: %w", err)
		}
		return cutoff, nil
	}
	if days < 0 {
		return time.Time{}, fmt.Errorf("--days must be non-negative")
	}
	return now.AddDate(0, 0, -days).Truncate(24 * time.Hour), nil
}

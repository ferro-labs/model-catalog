package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PruneSunset removes provider YAML files whose lifecycle.sunset_date is before cutoff.
func PruneSunset(providersDir string, cutoff time.Time, dryRun bool) (int, error) {
	pattern := filepath.Join(providersDir, "*/models/*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return 0, fmt.Errorf("glob: %w", err)
	}

	pruned := 0
	for _, path := range matches {
		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return 0, fmt.Errorf("read %s: %w", path, err)
		}

		entry, err := ReadModelYAML(data)
		if err != nil {
			return 0, fmt.Errorf("parse %s: %w", path, err)
		}
		if entry.Lifecycle.SunsetDate == nil || *entry.Lifecycle.SunsetDate == "" {
			continue
		}

		sunset, err := time.Parse("2006-01-02", *entry.Lifecycle.SunsetDate)
		if err != nil {
			return 0, fmt.Errorf("%s invalid sunset_date %q: %w", path, *entry.Lifecycle.SunsetDate, err)
		}
		if !sunset.Before(cutoff) {
			continue
		}

		fmt.Printf("prune %s/%s sunset_date=%s\n", entry.Provider, entry.ModelID, *entry.Lifecycle.SunsetDate)
		pruned++
		if dryRun {
			continue
		}
		if err := os.Remove(filepath.Clean(path)); err != nil {
			return 0, fmt.Errorf("remove %s: %w", path, err)
		}
	}

	return pruned, nil
}

package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BackfillSource fills empty source fields in provider YAML files using
// a pre-built map of catalog_key → source URL.
func BackfillSource(providersDir string, sourceMap map[string]string, dryRun bool) (int, error) {
	pattern := filepath.Join(providersDir, "*/models/*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return 0, fmt.Errorf("glob: %w", err)
	}

	updated := 0
	for _, path := range matches {
		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return 0, fmt.Errorf("read %s: %w", path, err)
		}

		content := string(data)
		if !strings.Contains(content, "source: \"\"") {
			continue
		}

		entry, err := ReadModelYAML(data)
		if err != nil {
			continue
		}

		key := entry.Provider + "/" + entry.ModelID
		sourceURL, ok := sourceMap[key]
		if !ok || sourceURL == "" {
			continue
		}

		if dryRun {
			fmt.Printf("[dry-run] %s → source: %q\n", key, sourceURL)
			updated++
			continue
		}

		newContent := strings.Replace(content, "source: \"\"", fmt.Sprintf("source: %q", sourceURL), 1)
		if err := os.WriteFile(filepath.Clean(path), []byte(newContent), 0o600); err != nil { //nolint:gosec // path from filepath.Glob
			return 0, fmt.Errorf("write %s: %w", path, err)
		}
		updated++
	}

	return updated, nil
}

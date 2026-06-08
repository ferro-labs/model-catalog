package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var providerDirPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// SanitizeFilename replaces characters that are unsafe in filenames.
func SanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "__")
	name = strings.ReplaceAll(name, ":", "_")
	return name
}

func validateProviderDir(provider string) error {
	if !providerDirPattern.MatchString(provider) {
		return fmt.Errorf("unsafe provider id %q", provider)
	}
	return nil
}

// Split reads a catalog JSON blob and writes per-model YAML files into
// outputDir organised as <provider>/models/<model>.yaml.
func Split(data []byte, outputDir string) error {
	entries, err := ReadCatalogJSON(data)
	if err != nil {
		return err
	}

	count := 0
	for key, entry := range entries {
		provider := entry.Provider
		modelID := entry.ModelID

		if provider == "" {
			parts := strings.SplitN(key, "/", 2)
			if len(parts) == 2 {
				provider = parts[0]
			} else {
				provider = "unknown"
			}
		}

		if modelID == "" {
			parts := strings.SplitN(key, "/", 2)
			if len(parts) == 2 {
				modelID = parts[1]
			} else {
				modelID = key
			}
		}

		if err := validateProviderDir(provider); err != nil {
			return fmt.Errorf("entry %s: %w", key, err)
		}

		dir := filepath.Join(outputDir, provider, "models")
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}

		filename := SanitizeFilename(modelID) + ".yaml"
		yamlData, err := WriteModelYAML(entry)
		if err != nil {
			return fmt.Errorf("marshal entry %s: %w", key, err)
		}

		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, yamlData, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		count++
	}

	fmt.Printf("Wrote %d model files to %s\n", count, outputDir)
	return nil
}

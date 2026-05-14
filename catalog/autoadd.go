package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AutoAddCandidate represents a model that can be auto-added to the catalog.
type AutoAddCandidate struct {
	Provider   string
	ModelID    string
	InputPerM  *float64
	OutputPerM *float64
	Sources    []string
}

// AutoAddResult summarizes what the auto-add operation did.
type AutoAddResult struct {
	Added      int
	Skipped    int
	NoProvider int
}

// AutoAdd writes YAML files for new models into providersDir.
// Only adds models where the provider folder already exists.
func AutoAdd(providersDir string, candidates []AutoAddCandidate, dryRun bool) (AutoAddResult, error) {
	var result AutoAddResult
	now := time.Now().UTC().Format("2006-01-02")

	for _, c := range candidates {
		providerDir := filepath.Join(providersDir, c.Provider, "models")
		if _, err := os.Stat(providerDir); os.IsNotExist(err) {
			result.NoProvider++
			continue
		}

		filename := SanitizeFilename(c.ModelID) + ".yaml"
		outPath := filepath.Join(providerDir, filename)

		if _, err := os.Stat(outPath); err == nil {
			result.Skipped++
			continue
		}

		if dryRun {
			fmt.Printf("[dry-run] Would add %s/%s (%s)\n", c.Provider, c.ModelID, strings.Join(c.Sources, "+"))
			result.Added++
			continue
		}

		data := buildAutoAddYAML(c, now)
		if err := os.WriteFile(outPath, data, 0o600); err != nil {
			return result, fmt.Errorf("write %s: %w", outPath, err)
		}
		result.Added++
	}

	return result, nil
}

func buildAutoAddYAML(c AutoAddCandidate, date string) []byte {
	var doc yaml.Node
	doc.Kind = yaml.DocumentNode

	mapping := &yaml.Node{Kind: yaml.MappingNode}
	doc.Content = append(doc.Content, mapping)

	AddStringField(mapping, "provider", c.Provider)
	AddStringField(mapping, "model_id", c.ModelID)
	AddStringField(mapping, "display_name", c.ModelID)
	AddStringField(mapping, "mode", "chat")
	AddIntField(mapping, "context_window", 0)
	AddIntField(mapping, "max_output_tokens", 0)

	pricing := &yaml.Node{Kind: yaml.MappingNode}
	addAutoAddPricing(pricing, "input_per_m_tokens", c.InputPerM)
	addAutoAddPricing(pricing, "output_per_m_tokens", c.OutputPerM)
	AddNullFloat64Field(pricing, "cache_read_per_m_tokens", NullFloat64{})
	AddNullFloat64Field(pricing, "cache_write_per_m_tokens", NullFloat64{})
	AddNullFloat64Field(pricing, "reasoning_per_m_tokens", NullFloat64{})
	AddNullFloat64Field(pricing, "image_per_tile", NullFloat64{})
	AddNullFloat64Field(pricing, "audio_input_per_minute", NullFloat64{})
	AddNullFloat64Field(pricing, "audio_output_per_character", NullFloat64{})
	AddNullFloat64Field(pricing, "embedding_per_m_tokens", NullFloat64{})
	AddNullFloat64Field(pricing, "finetune_train_per_m_tokens", NullFloat64{})
	AddNullFloat64Field(pricing, "finetune_input_per_m_tokens", NullFloat64{})
	AddNullFloat64Field(pricing, "finetune_output_per_m_tokens", NullFloat64{})
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "pricing"},
		pricing,
	)

	caps := CapabilitiesToYAML(Capabilities{})
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "capabilities"},
		caps,
	)

	lc := LifecycleToYAML(Lifecycle{Status: "preview"})
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "lifecycle"},
		lc,
	)

	AddStringField(mapping, "source", "auto:"+strings.Join(c.Sources, "+"))
	AddStringField(mapping, "updated_at", date)
	AddStringField(mapping, "tier", "standard")

	out, _ := yaml.Marshal(&doc)
	return out
}

func addAutoAddPricing(mapping *yaml.Node, key string, val *float64) {
	if val != nil {
		AddNullFloat64Field(mapping, key, NullFloat64{Value: *val, Valid: true})
	} else {
		AddNullFloat64Field(mapping, key, NullFloat64{})
	}
}

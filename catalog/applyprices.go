package catalog

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// PriceUpdate describes a single pricing-field change to apply to an existing
// model's YAML file. Field is the pricing sub-field name (e.g.
// "input_per_m_tokens"), without the "pricing." prefix.
type PriceUpdate struct {
	Provider string
	ModelID  string
	Field    string
	Value    float64
}

// ApplyResult summarizes an ApplyPriceUpdates run.
type ApplyResult struct {
	Applied    int      // number of individual field updates written
	Files      int      // number of distinct files changed
	NotApplied []string // "provider/model_id pricing.field" entries that could not be applied
}

// ApplyPriceUpdates surgically edits existing model YAML files in providersDir,
// updating the requested pricing fields in place. It preserves the rest of each
// file's structure (field order, comments, and the extends: key on wrappers) by
// editing the parsed YAML node tree rather than re-marshalling an Entry.
//
// Updates whose file or pricing field cannot be found are recorded in
// ApplyResult.NotApplied and skipped (never an error). When dryRun is true, no
// files are written.
func ApplyPriceUpdates(providersDir string, updates []PriceUpdate, date string, dryRun bool) (ApplyResult, error) {
	var result ApplyResult

	// Group updates by file so each file is read/written once.
	type fileGroup struct {
		path   string
		key    string
		fields []PriceUpdate
	}
	order := []string{}
	groups := map[string]*fileGroup{}
	for _, u := range updates {
		path := filepath.Join(providersDir, u.Provider, "models", SanitizeFilename(u.ModelID)+".yaml")
		g, ok := groups[path]
		if !ok {
			g = &fileGroup{path: path, key: u.Provider + "/" + u.ModelID}
			groups[path] = g
			order = append(order, path)
		}
		g.fields = append(g.fields, u)
	}

	for _, path := range order {
		g := groups[path]

		data, err := os.ReadFile(path)
		if err != nil {
			for _, u := range g.fields {
				result.NotApplied = append(result.NotApplied, g.key+" pricing."+u.Field)
			}
			continue
		}

		var doc yaml.Node
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return result, fmt.Errorf("parse %s: %w", path, err)
		}
		root := documentRoot(&doc)
		if root == nil {
			for _, u := range g.fields {
				result.NotApplied = append(result.NotApplied, g.key+" pricing."+u.Field)
			}
			continue
		}

		pricing := findMapValue(root, "pricing")
		appliedHere := 0
		for _, u := range g.fields {
			if pricing == nil {
				result.NotApplied = append(result.NotApplied, g.key+" pricing."+u.Field)
				continue
			}
			fieldNode := findMapValue(pricing, u.Field)
			if fieldNode == nil {
				result.NotApplied = append(result.NotApplied, g.key+" pricing."+u.Field)
				continue
			}
			setFloatScalar(fieldNode, u.Value)
			appliedHere++
		}

		if appliedHere == 0 {
			continue
		}

		setOrAddUpdatedAt(root, date)

		if dryRun {
			fmt.Printf("[dry-run] Would update %s (%d field(s))\n", g.key, appliedHere)
			result.Applied += appliedHere
			result.Files++
			continue
		}

		out, err := yaml.Marshal(&doc)
		if err != nil {
			return result, fmt.Errorf("marshal %s: %w", path, err)
		}
		if err := os.WriteFile(path, out, 0o600); err != nil {
			return result, fmt.Errorf("write %s: %w", path, err)
		}
		result.Applied += appliedHere
		result.Files++
	}

	return result, nil
}

// documentRoot returns the top-level mapping node of a parsed YAML document.
func documentRoot(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) == 1 {
		if doc.Content[0].Kind == yaml.MappingNode {
			return doc.Content[0]
		}
		return nil
	}
	if doc.Kind == yaml.MappingNode {
		return doc
	}
	return nil
}

// findMapValue returns the value node for key in a mapping node, or nil.
func findMapValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// setFloatScalar rewrites a scalar node to hold a plain float value, clearing
// any prior null tag/style so "null" becomes e.g. 0.2.
func setFloatScalar(node *yaml.Node, v float64) {
	node.Kind = yaml.ScalarNode
	node.Tag = ""
	node.Style = 0
	node.Value = fmt.Sprintf("%g", v)
}

// setOrAddUpdatedAt sets the top-level updated_at field (double-quoted to match
// the catalog convention), appending it if absent.
func setOrAddUpdatedAt(root *yaml.Node, date string) {
	if node := findMapValue(root, "updated_at"); node != nil {
		node.Kind = yaml.ScalarNode
		node.Tag = "!!str"
		node.Style = yaml.DoubleQuotedStyle
		node.Value = date
		return
	}
	root.Content = append(root.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "updated_at"},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.DoubleQuotedStyle, Value: date},
	)
}

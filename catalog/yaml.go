package catalog

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func ReadModelYAML(data []byte) (Entry, error) {
	var e Entry
	if err := yaml.Unmarshal(data, &e); err != nil {
		return Entry{}, fmt.Errorf("parse model YAML: %w", err)
	}
	return e, nil
}

func WriteModelYAML(e Entry) ([]byte, error) {
	data, err := yaml.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("marshal model YAML: %w", err)
	}
	return data, nil
}

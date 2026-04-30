package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

func ReadCatalogJSON(data []byte) (map[string]Entry, error) {
	var raw map[string]Entry
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse catalog JSON: %w", err)
	}
	return raw, nil
}

func WriteCatalogJSON(entries map[string]Entry) ([]byte, error) {
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteString("{\n")

	for i, k := range keys {
		keyJSON, err := json.Marshal(k)
		if err != nil {
			return nil, fmt.Errorf("marshal key %q: %w", k, err)
		}

		valJSON, err := json.MarshalIndent(entries[k], "  ", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal entry %q: %w", k, err)
		}

		buf.WriteString("  ")
		buf.Write(keyJSON)
		buf.WriteString(": ")
		buf.Write(valJSON)

		if i < len(keys)-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}

	buf.WriteString("}\n")
	return buf.Bytes(), nil
}

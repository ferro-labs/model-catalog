package catalog

import "testing"

func TestExtractCoreName(t *testing.T) {
	tests := []struct {
		modelID string
		vendor  string
		want    string
	}{
		{"anthropic.claude-3-haiku-20240307-v1:0", "anthropic", "claude-3-haiku-20240307"},
		{"us.anthropic.claude-3-haiku-20240307-v1:0", "anthropic", "claude-3-haiku-20240307"},
		{"meta.llama3-3-70b-instruct-v1:0", "meta", "llama3-3-70b-instruct"},
		{"us.meta.llama3-2-11b-instruct-v1:0", "meta", "llama3-2-11b-instruct"},
		{"meta.llama2-70b-chat-v1", "meta", "llama2-70b-chat-v1"},
		{"meta.llama4-maverick-17b-instruct-v1:0", "meta", "llama4-maverick-17b-instruct"},
		{"cohere.command-text-v14", "cohere", "command-text-v14"},
		{"no-match-here", "anthropic", ""},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			got := extractCoreName(tt.modelID, tt.vendor)
			if got != tt.want {
				t.Errorf("extractCoreName(%q, %q) = %q, want %q", tt.modelID, tt.vendor, got, tt.want)
			}
		})
	}
}

func TestNormalizeForFuzzy(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Llama-3.3-70B-Instruct", "llama3370binstruct"},
		{"llama3-3-70b-instruct", "llama3370binstruct"},
		{"Llama-4-Maverick-17B-128E-Instruct-FP8", "llama4maverick17b128einstructfp8"},
		{"llama4-maverick-17b-instruct", "llama4maverick17binstruct"},
		{"claude-3-haiku-20240307", "claude3haiku20240307"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeForFuzzy(tt.input)
			if got != tt.want {
				t.Errorf("normalizeForFuzzy(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFindBaseMatch_MetaLlama(t *testing.T) {
	baseEntries := map[string]Entry{
		"Llama-3.3-70B-Instruct":              {ModelID: "Llama-3.3-70B-Instruct"},
		"Llama-3.3-8B-Instruct":               {ModelID: "Llama-3.3-8B-Instruct"},
		"Llama-4-Maverick-17B-128E-Instruct-FP8": {ModelID: "Llama-4-Maverick-17B-128E-Instruct-FP8"},
		"Llama-4-Scout-17B-16E-Instruct-FP8":  {ModelID: "Llama-4-Scout-17B-16E-Instruct-FP8"},
	}

	tests := []struct {
		wrapperModelID string
		wantBaseID     string
		wantFound      bool
	}{
		{"meta.llama3-3-70b-instruct-v1:0", "Llama-3.3-70B-Instruct", true},
		{"us.meta.llama3-3-70b-instruct-v1:0", "Llama-3.3-70B-Instruct", true},
		{"meta.llama4-maverick-17b-instruct-v1:0", "Llama-4-Maverick-17B-128E-Instruct-FP8", true},
		{"meta.llama4-scout-17b-instruct-v1:0", "Llama-4-Scout-17B-16E-Instruct-FP8", true},
		{"meta.llama3-70b-instruct-v1:0", "", false},
		{"meta.llama3-8b-instruct-v1:0", "", false},
		{"meta.llama2-70b-chat-v1", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.wrapperModelID, func(t *testing.T) {
			gotID, _, gotFound := findBaseMatch(tt.wrapperModelID, baseEntries, "meta_llama")
			if gotFound != tt.wantFound {
				t.Errorf("findBaseMatch(%q) found=%v, want %v", tt.wrapperModelID, gotFound, tt.wantFound)
			}
			if gotFound && gotID != tt.wantBaseID {
				t.Errorf("findBaseMatch(%q) = %q, want %q", tt.wrapperModelID, gotID, tt.wantBaseID)
			}
		})
	}
}

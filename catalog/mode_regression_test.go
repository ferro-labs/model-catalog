package catalog

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func buildRepositoryCatalog(t *testing.T) map[string]Entry {
	t.Helper()
	distDir := t.TempDir()
	if err := BuildWithVersion(filepath.Join("..", "providers"), distDir, "v2026.08.22"); err != nil {
		t.Fatalf("BuildWithVersion() error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(distDir, "catalog.json"))
	if err != nil {
		t.Fatalf("read catalog.json: %v", err)
	}
	entries, err := ReadCatalogJSON(data)
	if err != nil {
		t.Fatalf("ReadCatalogJSON() error: %v", err)
	}
	return entries
}

func TestCatalogEndpointModeCorrections(t *testing.T) {
	entries := buildRepositoryCatalog(t)

	want := map[string]string{
		"gemini/deep-research-pro-preview-12-2025":                     "agent",
		"vertex_ai/deep-research-pro-preview-12-2025":                  "agent",
		"openai/o3-deep-research":                                      "responses",
		"azure/o3-deep-research":                                       "responses",
		"openai/gpt-5.3-codex":                                         "responses",
		"azure/global/gpt-5.1-codex":                                   "responses",
		"github_copilot/gpt-5.3-codex":                                 "chat",
		"openai/gpt-3.5-turbo-instruct":                                "completion",
		"azure/gpt-35-turbo-instruct":                                  "completion",
		"openai/gpt-realtime":                                          "realtime",
		"azure/gpt-realtime-2025-08-28":                                "realtime",
		"gemini/gemini-live-2.5-flash-preview-native-audio-09-2025":    "realtime",
		"vertex_ai/gemini-live-2.5-flash-preview-native-audio-09-2025": "realtime",
		"openai/sora-2":                                                "video",
		"azure/sora-2":                                                 "video",
		"gemini/veo-3.1-generate-001":                                  "video",
		"vertex_ai-video-models/vertex_ai/veo-3.1-generate-001":        "video",
		"runwayml/gen4_turbo":                                          "video",
		"openai/gpt-image-2":                                           "image",
		"azure/gpt-image-2":                                            "image",
		"gemini/gemini-2.0-flash-preview-image-generation":             "image",
		"stability/inpaint":                                            "image",
		"bedrock/stability.stable-image-inpaint-v1:0":                  "image",
		"gemini/gemini-2.5-pro-preview-tts":                            "audio_out",
		"mistral/mistral-ocr-latest":                                   "ocr",
		"vertex_ai/mistral-ocr-2505":                                   "ocr",
		"vercel_ai_gateway/cohere/embed-v4.0":                          "embedding",
		"fireworks/fireworks-ai-embedding-up-to-150m":                  "embedding",
		"openai/container":                                             "tool",
		"vertex_ai/search_api":                                         "tool",
	}

	for key, wantMode := range want {
		entry, ok := entries[key]
		if !ok {
			t.Errorf("missing catalog key %q", key)
			continue
		}
		if entry.Mode != wantMode {
			t.Errorf("%s mode = %q, want %q", key, entry.Mode, wantMode)
		}
	}
}

func TestCatalogEndpointFamilyInvariants(t *testing.T) {
	entries := buildRepositoryCatalog(t)

	type familyRule struct {
		name     string
		provider *regexp.Regexp
		model    *regexp.Regexp
		mode     string
	}
	rules := []familyRule{
		{"OpenAI realtime", regexp.MustCompile(`^(openai|azure)$`), regexp.MustCompile(`(^|/)gpt-(4o(-mini)?-)?realtime`), "realtime"},
		{"Gemini Live", regexp.MustCompile(`^(gemini|vertex_ai)$`), regexp.MustCompile(`gemini.*live`), "realtime"},
		{"OpenAI Sora", regexp.MustCompile(`^(openai|azure)$`), regexp.MustCompile(`^sora-2`), "video"},
		{"Gemini Veo", regexp.MustCompile(`^(gemini|vertex_ai-video-models)$`), regexp.MustCompile(`(^|/)veo-`), "video"},
		{"Runway video", regexp.MustCompile(`^runwayml$`), regexp.MustCompile(`^(gen3a_turbo|gen4_aleph|gen4_turbo)$`), "video"},
		{"Stability image services", regexp.MustCompile(`^stability$`), regexp.MustCompile(`.*`), "image"},
		{"Bedrock Stability image services", regexp.MustCompile(`^bedrock$`), regexp.MustCompile(`^stability\.`), "image"},
		{"Mistral OCR API", regexp.MustCompile(`^(mistral|vertex_ai)$`), regexp.MustCompile(`^mistral-ocr`), "ocr"},
	}

	for key, entry := range entries {
		for _, rule := range rules {
			if rule.provider.MatchString(entry.Provider) && rule.model.MatchString(entry.ModelID) && entry.Mode != rule.mode {
				t.Errorf("%s: %s mode = %q, want %q", rule.name, key, entry.Mode, rule.mode)
			}
		}
	}
}

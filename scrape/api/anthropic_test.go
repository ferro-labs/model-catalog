package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnthropicScrape(t *testing.T) {
	fixture := `{
		"data": [
			{"id": "claude-opus-4-7", "display_name": "Claude Opus 4.7", "type": "model"},
			{"id": "claude-sonnet-4-6", "display_name": "Claude Sonnet 4.6", "type": "model"},
			{"id": "claude-haiku-4-5-20251001", "display_name": "Claude Haiku 4.5", "type": "model"}
		],
		"has_more": false
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header, got %q", r.Header.Get("x-api-key"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer srv.Close()

	s := newAnthropicWithURL(srv.URL, srv.Client(), "test-key")
	obs, err := s.Scrape()
	if err != nil {
		t.Fatalf("Scrape() error: %v", err)
	}

	if len(obs) != 3 {
		t.Fatalf("got %d observations, want 3", len(obs))
	}

	want := map[string]bool{
		"claude-opus-4-7":           true,
		"claude-sonnet-4-6":         true,
		"claude-haiku-4-5-20251001": true,
	}
	for _, o := range obs {
		if !want[o.ModelID] {
			t.Errorf("unexpected model: %s", o.ModelID)
		}
		if o.Provider != "anthropic" {
			t.Errorf("got provider %q, want anthropic", o.Provider)
		}
		if o.Source != "anthropic_api" {
			t.Errorf("got source %q, want anthropic_api", o.Source)
		}
	}
}

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAICompatibleScrape(t *testing.T) {
	fixture := `{
		"object": "list",
		"data": [
			{"id": "llama-3.3-70b-versatile", "object": "model"},
			{"id": "openai/gpt-oss-120b", "object": "model"}
		]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("expected Bearer auth, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer srv.Close()

	s := newOpenAICompatible("groq", "groq_api", srv.URL, srv.Client(), "test-key")
	obs, err := s.Scrape()
	if err != nil {
		t.Fatalf("Scrape() error: %v", err)
	}

	if len(obs) != 2 {
		t.Fatalf("got %d observations, want 2", len(obs))
	}
	for _, o := range obs {
		if o.Provider != "groq" {
			t.Errorf("got provider %q, want groq", o.Provider)
		}
		if o.Source != "groq_api" {
			t.Errorf("got source %q, want groq_api", o.Source)
		}
	}
}

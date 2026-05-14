package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTogetherScrape(t *testing.T) {
	fixture := `[
		{"id": "meta-llama/Llama-3.3-70B-Instruct-Turbo"},
		{"id": "deepseek-ai/DeepSeek-R1"}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("expected Bearer auth, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer srv.Close()

	s := newTogetherWithURL(srv.URL, srv.Client(), "test-key")
	obs, err := s.Scrape()
	if err != nil {
		t.Fatalf("Scrape() error: %v", err)
	}

	if len(obs) != 2 {
		t.Fatalf("got %d observations, want 2", len(obs))
	}
	for _, o := range obs {
		if o.Provider != "together" {
			t.Errorf("got provider %q, want together", o.Provider)
		}
		if o.Source != "together_api" {
			t.Errorf("got source %q, want together_api", o.Source)
		}
	}
}

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCohereScrape(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("expected Bearer auth, got %q", auth)
		}
		requests++
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			_, _ = w.Write([]byte(`{"models":[{"name":"command-a-03-2025"}],"next_page_token":"next"}`))
			return
		}
		if got := r.URL.Query().Get("page_token"); got != "next" {
			t.Errorf("got page_token %q, want next", got)
		}
		_, _ = w.Write([]byte(`{"models":[{"name":"embed-v4.0"}]}`))
	}))
	defer srv.Close()

	s := newCohereWithURL(srv.URL, srv.Client(), "test-key")
	obs, err := s.Scrape()
	if err != nil {
		t.Fatalf("Scrape() error: %v", err)
	}

	if len(obs) != 2 {
		t.Fatalf("got %d observations, want 2", len(obs))
	}
	if requests != 2 {
		t.Fatalf("got %d requests, want 2", requests)
	}
	for _, o := range obs {
		if o.Provider != "cohere" {
			t.Errorf("got provider %q, want cohere", o.Provider)
		}
	}
}

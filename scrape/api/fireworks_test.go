package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFireworksScrape(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("expected Bearer auth, got %q", auth)
		}
		requests++
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			_, _ = w.Write([]byte(`{"models":[{"name":"accounts/fireworks/models/gpt-oss-120b"}],"nextPageToken":"next"}`))
			return
		}
		if got := r.URL.Query().Get("pageToken"); got != "next" {
			t.Errorf("got pageToken %q, want next", got)
		}
		_, _ = w.Write([]byte(`{"models":[{"name":"accounts/fireworks/models/deepseek-v3p2"}]}`))
	}))
	defer srv.Close()

	s := newFireworksWithURL(srv.URL, srv.Client(), "test-key")
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
		if o.Provider != "fireworks" {
			t.Errorf("got provider %q, want fireworks", o.Provider)
		}
		if o.Source != "fireworks_api" {
			t.Errorf("got source %q, want fireworks_api", o.Source)
		}
	}
}

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIScrape(t *testing.T) {
	fixture := `{
		"data": [
			{"id": "gpt-4o", "owned_by": "openai"},
			{"id": "gpt-4.1", "owned_by": "openai"},
			{"id": "o3-pro", "owned_by": "openai"}
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

	s := newOpenAIWithURL(srv.URL, srv.Client(), "test-key")
	obs, err := s.Scrape()
	if err != nil {
		t.Fatalf("Scrape() error: %v", err)
	}

	if len(obs) != 3 {
		t.Fatalf("got %d observations, want 3", len(obs))
	}

	want := map[string]bool{
		"gpt-4o":  true,
		"gpt-4.1": true,
		"o3-pro":  true,
	}
	for _, o := range obs {
		if !want[o.ModelID] {
			t.Errorf("unexpected model: %s", o.ModelID)
		}
		if o.Provider != "openai" {
			t.Errorf("got provider %q, want openai", o.Provider)
		}
	}
}

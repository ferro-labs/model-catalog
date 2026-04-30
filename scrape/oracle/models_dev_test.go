package oracle

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ferro-labs/model-catalog/scrape"
)

const modelsDevFixture = `{
  "openai": {
    "id": "openai",
    "models": {
      "gpt-4o": {
        "cost": {"input": 2.5, "output": 10.0, "cache_read": 1.25},
        "limit": {"context": 128000, "output": 16384}
      },
      "gpt-4o-mini": {
        "cost": {"input": 0.15, "output": 0.6},
        "limit": {"context": 128000, "output": 16384}
      }
    }
  },
  "anthropic": {
    "id": "anthropic",
    "models": {
      "claude-sonnet-4-5": {
        "cost": {"input": 3.0, "output": 15.0, "cache_read": 0.3},
        "limit": {"context": 200000, "output": 8192}
      }
    }
  },
  "meta-llama": {
    "id": "meta-llama",
    "models": {
      "llama-3.3-70b": {
        "cost": {"input": 0.0, "output": 0.0},
        "limit": {"context": 131072}
      }
    }
  }
}`

func TestModelsDevScrape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); ua != scrape.UserAgent {
			t.Errorf("unexpected User-Agent: %s", ua)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(modelsDevFixture))
	}))
	defer srv.Close()

	scraper := newModelsDevWithURL(srv.URL, srv.Client())
	obs, err := scraper.Scrape()
	if err != nil {
		t.Fatalf("Scrape() error: %v", err)
	}

	if len(obs) != 4 {
		t.Fatalf("expected 4 observations, got %d", len(obs))
	}

	// Verify openai/gpt-4o
	gpt4o := findObs(obs, "openai", "gpt-4o")
	if gpt4o == nil {
		t.Fatal("missing openai/gpt-4o")
	}
	assertFloatPtr(t, "input", gpt4o.InputPerM, 2.5)
	assertFloatPtr(t, "output", gpt4o.OutputPerM, 10.0)
	assertFloatPtr(t, "cache_read", gpt4o.CacheReadPerM, 1.25)
	assertIntPtr(t, "context", gpt4o.ContextWindow, 128000)
	assertIntPtr(t, "max_output", gpt4o.MaxOutput, 16384)

	// Verify anthropic/claude-sonnet-4-5
	claude := findObs(obs, "anthropic", "claude-sonnet-4-5")
	if claude == nil {
		t.Fatal("missing anthropic/claude-sonnet-4-5")
	}
	assertFloatPtr(t, "input", claude.InputPerM, 3.0)
	assertFloatPtr(t, "output", claude.OutputPerM, 15.0)
	assertFloatPtr(t, "cache_read", claude.CacheReadPerM, 0.3)

	// Verify model with no cache_read
	mini := findObs(obs, "openai", "gpt-4o-mini")
	if mini == nil {
		t.Fatal("missing openai/gpt-4o-mini")
	}
	if mini.CacheReadPerM != nil {
		t.Errorf("expected nil CacheReadPerM for gpt-4o-mini, got %f", *mini.CacheReadPerM)
	}

	// Verify model with no max_output
	llama := findObs(obs, "meta-llama", "llama-3.3-70b")
	if llama == nil {
		t.Fatal("missing meta-llama/llama-3.3-70b")
	}
	if llama.MaxOutput != nil {
		t.Errorf("expected nil MaxOutput for llama, got %d", *llama.MaxOutput)
	}
}

func TestModelsDevHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	scraper := newModelsDevWithURL(srv.URL, srv.Client())
	_, err := scraper.Scrape()
	if err == nil {
		t.Fatal("expected error for 503 response")
	}
}

func TestModelsDevMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer srv.Close()

	scraper := newModelsDevWithURL(srv.URL, srv.Client())
	_, err := scraper.Scrape()
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

package oracle

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ferro-labs/model-catalog/scrape"
)

const openRouterFixture = `{
  "data": [
    {
      "id": "openai/gpt-4o",
      "pricing": {"prompt": "0.0000025", "completion": "0.00001"},
      "context_length": 128000,
      "top_provider": {"max_completion_tokens": 16384}
    },
    {
      "id": "anthropic/claude-sonnet-4-5",
      "pricing": {"prompt": "0.000003", "completion": "0.000015"},
      "context_length": 200000,
      "top_provider": {"max_completion_tokens": 8192}
    },
    {
      "id": "google/gemini-2.0-flash",
      "pricing": {"prompt": "0", "completion": "0"},
      "context_length": 1000000,
      "top_provider": null
    },
    {
      "id": "deepinfra/meta-llama/Llama-3.3-70B-Instruct",
      "pricing": {"prompt": "0.00000035", "completion": "0.0000004"},
      "context_length": 131072,
      "top_provider": {"max_completion_tokens": 4096}
    },
    {
      "id": "malformed-no-slash",
      "pricing": {"prompt": "0.001", "completion": "0.001"},
      "context_length": 4096
    }
  ]
}`

func TestOpenRouterPriceConversion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		want    float64
	}{
		{"standard price", "0.0000025", false, 2.5},
		{"higher price", "0.00001", false, 10.0},
		{"free model", "0", false, 0.0},
		{"empty string", "", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertPerTokenToPerM(tt.input)
			if tt.wantNil {
				if err == nil {
					t.Errorf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if math.Abs(*got-tt.want) > 0.001 {
				t.Errorf("convertPerTokenToPerM(%q) = %f, want %f", tt.input, *got, tt.want)
			}
		})
	}
}

func TestOpenRouterSplitID(t *testing.T) {
	tests := []struct {
		id           string
		wantProvider string
		wantModel    string
	}{
		{"openai/gpt-4o", "openai", "gpt-4o"},
		{"anthropic/claude-sonnet-4-5", "anthropic", "claude-sonnet-4-5"},
		{"deepinfra/meta-llama/Llama-3.3-70B", "deepinfra", "meta-llama/Llama-3.3-70B"},
		{"malformed-no-slash", "", ""},
		{"trailing-slash/", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			provider, model := splitOpenRouterID(tt.id)
			if provider != tt.wantProvider || model != tt.wantModel {
				t.Errorf("splitOpenRouterID(%q) = (%q, %q), want (%q, %q)",
					tt.id, provider, model, tt.wantProvider, tt.wantModel)
			}
		})
	}
}

func TestOpenRouterScrape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); ua != scrape.UserAgent {
			t.Errorf("unexpected User-Agent: %s", ua)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(openRouterFixture))
	}))
	defer srv.Close()

	scraper := newOpenRouterWithURL(srv.URL, srv.Client())
	obs, err := scraper.Scrape()
	if err != nil {
		t.Fatalf("Scrape() error: %v", err)
	}

	// "malformed-no-slash" should be skipped
	if len(obs) != 4 {
		t.Fatalf("expected 4 observations, got %d", len(obs))
	}

	// Verify first model: openai/gpt-4o
	gpt4o := findObs(obs, "openai", "gpt-4o")
	if gpt4o == nil {
		t.Fatal("missing openai/gpt-4o")
	}
	assertFloatPtr(t, "input", gpt4o.InputPerM, 2.5)
	assertFloatPtr(t, "output", gpt4o.OutputPerM, 10.0)
	assertIntPtr(t, "context", gpt4o.ContextWindow, 128000)
	assertIntPtr(t, "max_output", gpt4o.MaxOutput, 16384)

	// Verify free model
	flash := findObs(obs, "google", "gemini-2.0-flash")
	if flash == nil {
		t.Fatal("missing google/gemini-2.0-flash")
	}
	assertFloatPtr(t, "input", flash.InputPerM, 0.0)
	assertFloatPtr(t, "output", flash.OutputPerM, 0.0)
	if flash.MaxOutput != nil {
		t.Errorf("expected nil MaxOutput for model without top_provider, got %d", *flash.MaxOutput)
	}

	// Verify nested ID: deepinfra/meta-llama/Llama-3.3-70B-Instruct
	llama := findObs(obs, "deepinfra", "meta-llama/Llama-3.3-70B-Instruct")
	if llama == nil {
		t.Fatal("missing deepinfra/meta-llama/Llama-3.3-70B-Instruct")
	}
	assertFloatPtr(t, "input", llama.InputPerM, 0.35)
	assertFloatPtr(t, "output", llama.OutputPerM, 0.4)
}

func TestOpenRouterHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	scraper := newOpenRouterWithURL(srv.URL, srv.Client())
	_, err := scraper.Scrape()
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

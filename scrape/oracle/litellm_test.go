package oracle

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const litellmFixture = `{
  "sample_spec": {"litellm_provider": "openai", "input_cost_per_token": 0.1},
  "gpt-4o": {
    "litellm_provider": "openai",
    "input_cost_per_token": 0.0000025,
    "output_cost_per_token": 0.00001,
    "cache_read_input_token_cost": 0.00000125,
    "mode": "chat"
  },
  "openai/o1-mini": {
    "litellm_provider": "openai",
    "input_cost_per_token": 0.0000011,
    "output_cost_per_token": 0.0000044
  },
  "claude-3-5-sonnet-20241022": {
    "litellm_provider": "anthropic",
    "input_cost_per_token": 0.000003,
    "output_cost_per_token": 0.000015
  },
  "no-provider-model": {
    "input_cost_per_token": 0.000001
  }
}`

func litellmObs(t *testing.T, body string) map[string]scrapeObs {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	obs, err := newLiteLLMWithURL(srv.URL, srv.Client()).Scrape()
	if err != nil {
		t.Fatalf("Scrape: %v", err)
	}
	byKey := make(map[string]scrapeObs)
	for _, o := range obs {
		byKey[o.Provider+"/"+o.ModelID] = scrapeObs{o.InputPerM, o.OutputPerM, o.CacheReadPerM}
	}
	return byKey
}

type scrapeObs struct {
	in, out, cacheRead *float64
}

func TestLiteLLMScrape(t *testing.T) {
	byKey := litellmObs(t, litellmFixture)

	// sample_spec and provider-less entries are skipped.
	if _, ok := byKey["openai/sample_spec"]; ok {
		t.Error("sample_spec should be skipped")
	}
	if len(byKey) != 3 {
		t.Fatalf("got %d observations, want 3 (gpt-4o, o1-mini, claude); keys=%v", len(byKey), keysOf(byKey))
	}

	// Per-token costs convert to per-1M (×1e6).
	got := byKey["openai/gpt-4o"]
	if got.in == nil || *got.in != 2.5 {
		t.Errorf("gpt-4o input = %v, want 2.5", deref(got.in))
	}
	if got.out == nil || *got.out != 10 {
		t.Errorf("gpt-4o output = %v, want 10", deref(got.out))
	}
	if got.cacheRead == nil || *got.cacheRead != 1.25 {
		t.Errorf("gpt-4o cache_read = %v, want 1.25", deref(got.cacheRead))
	}

	// Provider prefix is stripped from the key.
	if _, ok := byKey["openai/o1-mini"]; !ok {
		t.Errorf("expected openai/o1-mini (prefix stripped); keys=%v", keysOf(byKey))
	}

	// Missing cache_read stays nil, not 0.
	if o := byKey["openai/o1-mini"]; o.cacheRead != nil {
		t.Errorf("o1-mini cache_read = %v, want nil", deref(o.cacheRead))
	}
}

func deref(f *float64) float64 {
	if f == nil {
		return -1
	}
	return *f
}

func keysOf(m map[string]scrapeObs) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

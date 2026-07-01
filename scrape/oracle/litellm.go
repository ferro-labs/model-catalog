package oracle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ferro-labs/model-catalog/scrape"
)

const litellmURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

// litellmTokensPerM converts LiteLLM's per-token costs to the catalog's
// per-1,000,000-token convention.
const litellmTokensPerM = 1_000_000

// litellmModel is one entry in LiteLLM's model_prices_and_context_window.json.
// Costs are USD per single token.
type litellmModel struct {
	Provider              string   `json:"litellm_provider"`
	InputCostPerToken     *float64 `json:"input_cost_per_token"`
	OutputCostPerToken    *float64 `json:"output_cost_per_token"`
	CacheReadCostPerToken *float64 `json:"cache_read_input_token_cost"`
}

// LiteLLM scrapes model pricing from the community-maintained LiteLLM price map.
// It is a free, keyless third corroborating oracle alongside OpenRouter and
// models.dev, so a price change confirmed by two of the three auto-applies.
type LiteLLM struct {
	client *http.Client
	url    string
}

// NewLiteLLM creates a LiteLLM scraper with the default endpoint.
func NewLiteLLM() *LiteLLM { return &LiteLLM{client: scrape.DefaultClient, url: litellmURL} }

func newLiteLLMWithURL(url string, client *http.Client) *LiteLLM {
	return &LiteLLM{client: client, url: url}
}

func (l *LiteLLM) Name() string { return "litellm" }

func (l *LiteLLM) Scrape() ([]scrape.Observation, error) {
	body, err := scrape.FetchJSON(l.client, l.url)
	if err != nil {
		return nil, fmt.Errorf("litellm: %w", err)
	}
	return parseLiteLLMResponse(body)
}

func parseLiteLLMResponse(data []byte) ([]scrape.Observation, error) {
	var models map[string]litellmModel
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, fmt.Errorf("litellm: parse JSON: %w", err)
	}

	now := time.Now().UTC()
	var observations []scrape.Observation
	for key, m := range models {
		// "sample_spec" is a documentation placeholder; entries without a
		// provider can't be matched to a catalog key.
		if key == "sample_spec" || strings.TrimSpace(m.Provider) == "" {
			continue
		}
		observations = append(observations, scrape.Observation{
			Source:        "litellm",
			SourceURL:     litellmURL,
			ScrapedAt:     now,
			Provider:      m.Provider,
			ModelID:       stripProviderPrefix(key, m.Provider),
			InputPerM:     perMFromPerToken(m.InputCostPerToken),
			OutputPerM:    perMFromPerToken(m.OutputCostPerToken),
			CacheReadPerM: perMFromPerToken(m.CacheReadCostPerToken),
		})
	}
	return observations, nil
}

// perMFromPerToken converts a per-token cost to per-1M tokens, preserving nil.
func perMFromPerToken(v *float64) *float64 {
	if v == nil {
		return nil
	}
	perM := *v * litellmTokensPerM
	return &perM
}

// stripProviderPrefix removes a leading "<provider>/" from a LiteLLM key so the
// model ID lines up with the catalog's per-provider model_id.
func stripProviderPrefix(key, provider string) string {
	return strings.TrimPrefix(key, provider+"/")
}

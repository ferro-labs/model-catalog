package oracle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ferro-labs/model-catalog/scrape"
)

const openRouterURL = "https://openrouter.ai/api/v1/models"

type openRouterResponse struct {
	Data []openRouterModel `json:"data"`
}

type openRouterModel struct {
	ID            string             `json:"id"`
	Pricing       openRouterPricing  `json:"pricing"`
	ContextLength int                `json:"context_length"`
	TopProvider   *openRouterTopProv `json:"top_provider"`
}

type openRouterPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}

type openRouterTopProv struct {
	MaxCompletionTokens *int `json:"max_completion_tokens"`
}

// OpenRouter scrapes model data from openrouter.ai.
type OpenRouter struct {
	client *http.Client
	url    string
}

// NewOpenRouter creates an OpenRouter scraper with the default endpoint.
func NewOpenRouter() *OpenRouter {
	return &OpenRouter{client: scrape.DefaultClient, url: openRouterURL}
}

func newOpenRouterWithURL(url string, client *http.Client) *OpenRouter {
	return &OpenRouter{client: client, url: url}
}

func (o *OpenRouter) Name() string { return "openrouter" }

func (o *OpenRouter) Scrape() ([]scrape.Observation, error) {
	body, err := scrape.FetchJSON(o.client, o.url)
	if err != nil {
		return nil, fmt.Errorf("openrouter: %w", err)
	}
	return parseOpenRouterResponse(body)
}

func parseOpenRouterResponse(data []byte) ([]scrape.Observation, error) {
	var resp openRouterResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("openrouter: parse JSON: %w", err)
	}

	now := time.Now().UTC()
	var observations []scrape.Observation

	for _, model := range resp.Data {
		provider, modelID := splitOpenRouterID(model.ID)
		if provider == "" || modelID == "" {
			continue
		}

		obs := scrape.Observation{
			Source:    "openrouter",
			SourceURL: openRouterURL,
			ScrapedAt: now,
			Provider:  provider,
			ModelID:   modelID,
		}

		if input, err := convertPerTokenToPerM(model.Pricing.Prompt); err == nil {
			obs.InputPerM = input
		}
		if output, err := convertPerTokenToPerM(model.Pricing.Completion); err == nil {
			obs.OutputPerM = output
		}

		if model.ContextLength > 0 {
			ctx := model.ContextLength
			obs.ContextWindow = &ctx
		}

		if model.TopProvider != nil && model.TopProvider.MaxCompletionTokens != nil {
			maxOut := *model.TopProvider.MaxCompletionTokens
			if maxOut > 0 {
				obs.MaxOutput = &maxOut
			}
		}

		observations = append(observations, obs)
	}

	return observations, nil
}

// splitOpenRouterID splits "openai/gpt-4o" into ("openai", "gpt-4o").
// For nested IDs like "deepinfra/meta-llama/Llama-3.3-70B", it splits on the first slash.
func splitOpenRouterID(id string) (string, string) {
	idx := strings.Index(id, "/")
	if idx < 0 || idx == len(id)-1 {
		return "", ""
	}
	return id[:idx], id[idx+1:]
}

// convertPerTokenToPerM converts a per-token price string to a *float64 in per-1M-token units.
// Returns nil for empty or unparseable values.
func convertPerTokenToPerM(s string) (*float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty price")
	}

	perToken, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, fmt.Errorf("parse price %q: %w", s, err)
	}

	perM := perToken * 1_000_000
	return &perM, nil
}

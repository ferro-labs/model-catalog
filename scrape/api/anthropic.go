package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ferro-labs/model-catalog/scrape"
)

const anthropicModelsURL = "https://api.anthropic.com/v1/models?limit=100"

type anthropicListResponse struct {
	Data []anthropicModel `json:"data"`
}

type anthropicModel struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

type Anthropic struct {
	client *http.Client
	url    string
	apiKey string
}

func NewAnthropic(apiKey string) *Anthropic {
	return &Anthropic{client: scrape.DefaultClient, url: anthropicModelsURL, apiKey: apiKey}
}

func newAnthropicWithURL(url string, client *http.Client, apiKey string) *Anthropic {
	return &Anthropic{client: client, url: url, apiKey: apiKey}
}

func (a *Anthropic) Name() string { return "anthropic_api" }

func (a *Anthropic) Scrape() ([]scrape.Observation, error) {
	body, err := a.fetch()
	if err != nil {
		return nil, fmt.Errorf("anthropic_api: %w", err)
	}
	return parseAnthropicResponse(body)
}

func (a *Anthropic) fetch() ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, a.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", scrape.UserAgent)
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var buf []byte
	buf, err = readBody(resp)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func parseAnthropicResponse(data []byte) ([]scrape.Observation, error) {
	var resp anthropicListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	now := time.Now().UTC()
	var observations []scrape.Observation

	for _, m := range resp.Data {
		observations = append(observations, scrape.Observation{
			Source:    "anthropic_api",
			SourceURL: "https://api.anthropic.com/v1/models",
			ScrapedAt: now,
			Provider:  "anthropic",
			ModelID:   m.ID,
		})
	}

	return observations, nil
}

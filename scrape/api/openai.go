package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ferro-labs/model-catalog/scrape"
)

const openaiModelsURL = "https://api.openai.com/v1/models"

type openaiListResponse struct {
	Data []openaiModel `json:"data"`
}

type openaiModel struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by"`
}

type OpenAI struct {
	client *http.Client
	url    string
	apiKey string
}

func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{client: scrape.DefaultClient, url: openaiModelsURL, apiKey: apiKey}
}

func newOpenAIWithURL(url string, client *http.Client, apiKey string) *OpenAI {
	return &OpenAI{client: client, url: url, apiKey: apiKey}
}

func (o *OpenAI) Name() string { return "openai_api" }

func (o *OpenAI) Scrape() ([]scrape.Observation, error) {
	body, err := o.fetch()
	if err != nil {
		return nil, fmt.Errorf("openai_api: %w", err)
	}
	return parseOpenAIResponse(body)
}

func (o *OpenAI) fetch() ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, o.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", scrape.UserAgent)
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return readBody(resp)
}

func parseOpenAIResponse(data []byte) ([]scrape.Observation, error) {
	var resp openaiListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	now := time.Now().UTC()
	var observations []scrape.Observation

	for _, m := range resp.Data {
		observations = append(observations, scrape.Observation{
			Source:    "openai_api",
			SourceURL: "https://api.openai.com/v1/models",
			ScrapedAt: now,
			Provider:  "openai",
			ModelID:   m.ID,
		})
	}

	return observations, nil
}

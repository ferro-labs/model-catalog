package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ferro-labs/model-catalog/scrape"
)

const togetherModelsURL = "https://api.together.ai/v1/models"

type togetherModel struct {
	ID string `json:"id"`
}

type Together struct {
	client *http.Client
	url    string
	apiKey string
}

func NewTogether(apiKey string) *Together {
	return &Together{client: scrape.DefaultClient, url: togetherModelsURL, apiKey: apiKey}
}

func newTogetherWithURL(url string, client *http.Client, apiKey string) *Together {
	return &Together{client: client, url: url, apiKey: apiKey}
}

func (t *Together) Name() string { return "together_api" }

func (t *Together) Scrape() ([]scrape.Observation, error) {
	body, err := t.fetch()
	if err != nil {
		return nil, fmt.Errorf("together_api: %w", err)
	}
	return parseTogetherResponse(body, t.url)
}

func (t *Together) fetch() ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, t.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", scrape.UserAgent)
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return readBody(resp)
}

func parseTogetherResponse(data []byte, sourceURL string) ([]scrape.Observation, error) {
	var models []togetherModel
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	now := time.Now().UTC()
	observations := make([]scrape.Observation, 0, len(models))
	for _, m := range models {
		if m.ID == "" {
			continue
		}
		observations = append(observations, scrape.Observation{
			Source:    "together_api",
			SourceURL: sourceURL,
			ScrapedAt: now,
			Provider:  "together",
			ModelID:   m.ID,
		})
	}
	return observations, nil
}

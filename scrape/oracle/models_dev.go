package oracle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ferro-labs/model-catalog/scrape"
)

const modelsDevURL = "https://models.dev/api.json"

// modelsDevProvider is the top-level provider object in the models.dev response.
type modelsDevProvider struct {
	ID     string                   `json:"id"`
	Models map[string]modelsDevModel `json:"models"`
}

type modelsDevModel struct {
	Cost  *modelsDevCost  `json:"cost"`
	Limit *modelsDevLimit `json:"limit"`
}

type modelsDevCost struct {
	Input     *float64 `json:"input"`
	Output    *float64 `json:"output"`
	CacheRead *float64 `json:"cache_read"`
}

type modelsDevLimit struct {
	Context *int `json:"context"`
	Output  *int `json:"output"`
}

// ModelsDev scrapes model data from models.dev.
type ModelsDev struct {
	client *http.Client
	url    string
}

// NewModelsDev creates a ModelsDev scraper with the default endpoint.
func NewModelsDev() *ModelsDev {
	return &ModelsDev{client: scrape.DefaultClient, url: modelsDevURL}
}

func newModelsDevWithURL(url string, client *http.Client) *ModelsDev {
	return &ModelsDev{client: client, url: url}
}

func (m *ModelsDev) Name() string { return "models_dev" }

func (m *ModelsDev) Scrape() ([]scrape.Observation, error) {
	body, err := scrape.FetchJSON(m.client, m.url)
	if err != nil {
		return nil, fmt.Errorf("models_dev: %w", err)
	}
	return parseModelsDevResponse(body)
}

func parseModelsDevResponse(data []byte) ([]scrape.Observation, error) {
	var providers map[string]modelsDevProvider
	if err := json.Unmarshal(data, &providers); err != nil {
		return nil, fmt.Errorf("models_dev: parse JSON: %w", err)
	}

	now := time.Now().UTC()
	var observations []scrape.Observation

	for providerID, provider := range providers {
		for modelID, model := range provider.Models {
			obs := scrape.Observation{
				Source:    "models_dev",
				SourceURL: modelsDevURL,
				ScrapedAt: now,
				Provider:  providerID,
				ModelID:   modelID,
			}

			if model.Cost != nil {
				obs.InputPerM = model.Cost.Input
				obs.OutputPerM = model.Cost.Output
				obs.CacheReadPerM = model.Cost.CacheRead
			}

			if model.Limit != nil {
				obs.ContextWindow = model.Limit.Context
				obs.MaxOutput = model.Limit.Output
			}

			observations = append(observations, obs)
		}
	}

	return observations, nil
}

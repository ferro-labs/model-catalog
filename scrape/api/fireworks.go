package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ferro-labs/model-catalog/scrape"
)

const fireworksModelsURL = "https://api.fireworks.ai/v1/accounts/%s/models"

type fireworksListResponse struct {
	Models        []fireworksModel `json:"models"`
	NextPageToken string           `json:"nextPageToken"`
}

type fireworksModel struct {
	Name string `json:"name"`
}

type Fireworks struct {
	client    *http.Client
	url       string
	apiKey    string
	accountID string
}

func NewFireworks(apiKey, accountID string) *Fireworks {
	if accountID == "" {
		accountID = "fireworks"
	}
	return &Fireworks{
		client:    scrape.DefaultClient,
		url:       fmt.Sprintf(fireworksModelsURL, accountID),
		apiKey:    apiKey,
		accountID: accountID,
	}
}

func newFireworksWithURL(url string, client *http.Client, apiKey string) *Fireworks {
	return &Fireworks{client: client, url: url, apiKey: apiKey}
}

func (f *Fireworks) Name() string { return "fireworks_api" }

func (f *Fireworks) Scrape() ([]scrape.Observation, error) {
	var out []scrape.Observation
	pageToken := ""
	for {
		body, err := f.fetch(pageToken)
		if err != nil {
			return nil, fmt.Errorf("fireworks_api: %w", err)
		}

		obs, next, err := parseFireworksResponse(body, f.url)
		if err != nil {
			return nil, err
		}
		out = append(out, obs...)

		if next == "" {
			return out, nil
		}
		pageToken = next
	}
}

func (f *Fireworks) fetch(pageToken string) ([]byte, error) {
	reqURL, err := url.Parse(f.url)
	if err != nil {
		return nil, err
	}
	q := reqURL.Query()
	q.Set("pageSize", "200")
	if pageToken != "" {
		q.Set("pageToken", pageToken)
	}
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", scrape.UserAgent)
	req.Header.Set("Authorization", "Bearer "+f.apiKey)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return readBody(resp)
}

func parseFireworksResponse(data []byte, sourceURL string) ([]scrape.Observation, string, error) {
	var resp fireworksListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, "", fmt.Errorf("parse: %w", err)
	}

	now := time.Now().UTC()
	observations := make([]scrape.Observation, 0, len(resp.Models))
	for _, m := range resp.Models {
		modelID := strings.TrimPrefix(m.Name, "models/")
		if modelID == "" {
			continue
		}
		observations = append(observations, scrape.Observation{
			Source:    "fireworks_api",
			SourceURL: sourceURL,
			ScrapedAt: now,
			Provider:  "fireworks",
			ModelID:   modelID,
		})
	}
	return observations, resp.NextPageToken, nil
}

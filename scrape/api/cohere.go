package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ferro-labs/model-catalog/scrape"
)

const cohereModelsURL = "https://api.cohere.com/v1/models"

type cohereListResponse struct {
	Models        []cohereModel `json:"models"`
	NextPageToken string        `json:"next_page_token"`
}

type cohereModel struct {
	Name string `json:"name"`
}

type Cohere struct {
	client *http.Client
	url    string
	apiKey string
}

func NewCohere(apiKey string) *Cohere {
	return &Cohere{client: scrape.DefaultClient, url: cohereModelsURL, apiKey: apiKey}
}

func newCohereWithURL(url string, client *http.Client, apiKey string) *Cohere {
	return &Cohere{client: client, url: url, apiKey: apiKey}
}

func (c *Cohere) Name() string { return "cohere_api" }

func (c *Cohere) Scrape() ([]scrape.Observation, error) {
	var out []scrape.Observation
	pageToken := ""
	for {
		body, err := c.fetch(pageToken)
		if err != nil {
			return nil, fmt.Errorf("cohere_api: %w", err)
		}

		obs, next, err := parseCohereResponse(body, c.url)
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

func (c *Cohere) fetch(pageToken string) ([]byte, error) {
	reqURL, err := url.Parse(c.url)
	if err != nil {
		return nil, err
	}
	q := reqURL.Query()
	q.Set("page_size", "1000")
	if pageToken != "" {
		q.Set("page_token", pageToken)
	}
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", scrape.UserAgent)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return readBody(resp)
}

func parseCohereResponse(data []byte, sourceURL string) ([]scrape.Observation, string, error) {
	var resp cohereListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, "", fmt.Errorf("parse: %w", err)
	}

	now := time.Now().UTC()
	observations := make([]scrape.Observation, 0, len(resp.Models))
	for _, m := range resp.Models {
		if m.Name == "" {
			continue
		}
		observations = append(observations, scrape.Observation{
			Source:    "cohere_api",
			SourceURL: sourceURL,
			ScrapedAt: now,
			Provider:  "cohere",
			ModelID:   m.Name,
		})
	}
	return observations, resp.NextPageToken, nil
}

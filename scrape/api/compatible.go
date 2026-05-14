package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ferro-labs/model-catalog/scrape"
)

type openAICompatibleListResponse struct {
	Data []openAICompatibleModel `json:"data"`
}

type openAICompatibleModel struct {
	ID string `json:"id"`
}

type OpenAICompatible struct {
	client   *http.Client
	url      string
	apiKey   string
	provider string
	source   string
}

func NewGroq(apiKey string) *OpenAICompatible {
	return newOpenAICompatible("groq", "groq_api", "https://api.groq.com/openai/v1/models", scrape.DefaultClient, apiKey)
}

func NewMistral(apiKey string) *OpenAICompatible {
	return newOpenAICompatible("mistral", "mistral_api", "https://api.mistral.ai/v1/models", scrape.DefaultClient, apiKey)
}

func NewDeepSeek(apiKey string) *OpenAICompatible {
	return newOpenAICompatible("deepseek", "deepseek_api", "https://api.deepseek.com/models", scrape.DefaultClient, apiKey)
}

func NewXAI(apiKey string) *OpenAICompatible {
	return newOpenAICompatible("xai", "xai_api", "https://api.x.ai/v1/models", scrape.DefaultClient, apiKey)
}

func NewCerebras(apiKey string) *OpenAICompatible {
	return newOpenAICompatible("cerebras", "cerebras_api", "https://api.cerebras.ai/v1/models", scrape.DefaultClient, apiKey)
}

func newOpenAICompatible(provider, source, url string, client *http.Client, apiKey string) *OpenAICompatible {
	return &OpenAICompatible{
		client:   client,
		url:      url,
		apiKey:   apiKey,
		provider: provider,
		source:   source,
	}
}

func (o *OpenAICompatible) Name() string { return o.source }

func (o *OpenAICompatible) Scrape() ([]scrape.Observation, error) {
	body, err := o.fetch()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.source, err)
	}
	return parseOpenAICompatibleResponse(body, o.provider, o.source, o.url)
}

func (o *OpenAICompatible) fetch() ([]byte, error) {
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

func parseOpenAICompatibleResponse(data []byte, provider, source, sourceURL string) ([]scrape.Observation, error) {
	var resp openAICompatibleListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	now := time.Now().UTC()
	observations := make([]scrape.Observation, 0, len(resp.Data))
	for _, m := range resp.Data {
		if m.ID == "" {
			continue
		}
		observations = append(observations, scrape.Observation{
			Source:    source,
			SourceURL: sourceURL,
			ScrapedAt: now,
			Provider:  provider,
			ModelID:   m.ID,
		})
	}

	return observations, nil
}

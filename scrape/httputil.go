package scrape

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	UserAgent      = "ferro-labs-ai-catalog-scraper/1.0 (+https://github.com/ferro-labs/model-catalog)"
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
	retryBaseDelay = 2 * time.Second
)

var DefaultClient = &http.Client{Timeout: defaultTimeout}

// FetchJSON performs a GET request with retries on transient failures (5xx, timeouts).
// Returns the response body bytes. Caller is responsible for JSON decoding.
func FetchJSON(client *http.Client, url string) ([]byte, error) {
	if client == nil {
		client = DefaultClient
	}

	var lastErr error
	for attempt := range maxRetries {
		body, err := doGet(client, url)
		if err == nil {
			return body, nil
		}
		lastErr = err

		if !isRetryable(err) {
			return nil, err
		}

		if attempt < maxRetries-1 {
			time.Sleep(retryBaseDelay * time.Duration(1<<uint(attempt)))
		}
	}
	return nil, fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}

func doGet(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 500 {
		return nil, &retryableError{status: resp.StatusCode, url: url}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", url, err)
	}
	return body, nil
}

type retryableError struct {
	status int
	url    string
}

func (e *retryableError) Error() string {
	return fmt.Sprintf("fetch %s: HTTP %d (retryable)", e.url, e.status)
}

func isRetryable(err error) bool {
	if _, ok := err.(*retryableError); ok {
		return true
	}
	return false
}

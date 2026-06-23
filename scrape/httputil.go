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
		// Connection resets, timeouts, and DNS hiccups are transient on CI
		// runners; retry rather than fail the whole scrape on one blip.
		return nil, &retryableError{msg: fmt.Sprintf("fetch %s: %v", url, err), err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 500 {
		return nil, &retryableError{msg: fmt.Sprintf("fetch %s: HTTP %d (retryable)", url, resp.StatusCode)}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// A read failure mid-body is a dropped connection — also transient.
		return nil, &retryableError{msg: fmt.Sprintf("read %s: %v", url, err), err: err}
	}
	return body, nil
}

type retryableError struct {
	msg string
	err error
}

func (e *retryableError) Error() string { return e.msg }

func (e *retryableError) Unwrap() error { return e.err }

func isRetryable(err error) bool {
	if _, ok := err.(*retryableError); ok {
		return true
	}
	return false
}

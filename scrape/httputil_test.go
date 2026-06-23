package scrape

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestFetchJSONSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != UserAgent {
			t.Errorf("User-Agent = %q, want %q", r.Header.Get("User-Agent"), UserAgent)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	body, err := FetchJSON(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("FetchJSON: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q", body)
	}
}

func TestFetchJSON404NoRetry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(404)
	}))
	defer srv.Close()

	_, err := FetchJSON(srv.Client(), srv.URL)
	if err == nil {
		t.Fatal("expected error on 404")
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1 (no retry on 4xx)", calls.Load())
	}
}

func TestFetchJSON500Retries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(503)
			return
		}
		_, _ = w.Write([]byte(`{"recovered":true}`))
	}))
	defer srv.Close()

	body, err := FetchJSON(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("FetchJSON: %v", err)
	}
	if string(body) != `{"recovered":true}` {
		t.Errorf("body = %q", body)
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}

func TestFetchJSONRetriesNetworkError(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			// Drop the connection mid-request to simulate a transient
			// network failure (reset/EOF) rather than an HTTP status.
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("ResponseWriter is not a Hijacker")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack: %v", err)
			}
			_ = conn.Close()
			return
		}
		_, _ = w.Write([]byte(`{"recovered":true}`))
	}))
	defer srv.Close()

	body, err := FetchJSON(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("FetchJSON: %v", err)
	}
	if string(body) != `{"recovered":true}` {
		t.Errorf("body = %q", body)
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3 (network errors should be retried)", calls.Load())
	}
}

func TestFetchJSONAllRetriesFail(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(500)
	}))
	defer srv.Close()

	_, err := FetchJSON(srv.Client(), srv.URL)
	if err == nil {
		t.Fatal("expected error after all retries exhausted")
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3 (maxRetries)", calls.Load())
	}
}

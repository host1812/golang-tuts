package checker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// roundTripFunc lets tests supply a fake HTTP transport without network access.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCheckHandler(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		code := http.StatusNoContent
		if r.URL.Host == "bad.test" {
			code = http.StatusServiceUnavailable
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /check", Handler(client, []string{"https://ok.test", "https://bad.test", "://invalid"}))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/check", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response checkResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Sites) != 3 || !response.Sites[0].Available || response.Sites[0].StatusCode != 204 || response.Sites[1].Available || response.Sites[1].StatusCode != 503 || response.Sites[2].Error == "" {
		t.Fatalf("unexpected results: %+v", response.Sites)
	}

	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/check", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want 405", recorder.Code)
	}
}

func TestCheckSiteCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := checkSite(ctx, &http.Client{Timeout: time.Second}, "https://example.com")
	if got.Available || got.Error == "" {
		t.Fatalf("expected cancellation error, got %+v", got)
	}
}

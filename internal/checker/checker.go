// Package checker implements the website availability endpoint.
package checker

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

// Keep the targets on the server rather than accepting arbitrary URLs from a
// caller. This also prevents callers from using the service to probe internal
// addresses (a common server-side request forgery, or SSRF, risk).
var Sites = []string{
	"https://google.com", "https://microsoft.com", "https://amazom.com",
	"https://yahoo.com", "https://cnn.com", "https://weather.com",
}

// A result describes what this service observed, not whether a website is
// accessible to every user on the internet.
type result struct {
	URL        string `json:"url"`
	Available  bool   `json:"available"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

type checkResponse struct {
	Sites []result `json:"sites"`
}

// checkHandler accepts its dependencies as arguments. This makes the handler
// easy to test with local HTTP servers, without contacting real websites.
func Handler(client *http.Client, targets []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Checks can finish in any order, so send the original index along
		// with each result to preserve the configured order in the response.
		type indexedResult struct {
			index  int
			result result
		}
		results := make([]result, len(targets))
		// A buffered channel holds one result per worker. Workers can finish
		// their sends even if the handler has not started receiving yet.
		resultsCh := make(chan indexedResult, len(targets))
		for i, target := range targets {
			go func(i int, target string) {
				siteResult := checkSite(r.Context(), client, target)
				// The arrow points toward the channel when sending a value.
				resultsCh <- indexedResult{index: i, result: siteResult}
			}(i, target)
		}
		// Receive exactly one result per target. Each receive waits until a
		// result is available; only this goroutine writes to the slice.
		for range targets {
			received := <-resultsCh
			results[received.index] = received.result
		}
		// We know exactly how many values to receive, so closing the channel
		// is unnecessary. It will be garbage collected when no longer used.

		// If the caller has disconnected, there is no response to deliver.
		if r.Context().Err() != nil {
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(checkResponse{Sites: results}); err != nil {
			log.Printf("encode response: %v", err)
		}
	}
}

func checkSite(ctx context.Context, client *http.Client, target string) result {
	out := result{URL: target}
	// The incoming request's context cancels outgoing requests if the caller
	// disconnects. The client's timeout separately bounds each site's check.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	resp, err := client.Do(req)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	// Always close response bodies, even when we do not need their contents.
	defer resp.Body.Close()
	out.StatusCode = resp.StatusCode
	// The client follows redirects by default. A final 2xx or 3xx response
	// means an HTTP server responded successfully for this simple check.
	out.Available = resp.StatusCode >= 200 && resp.StatusCode < 400
	return out
}

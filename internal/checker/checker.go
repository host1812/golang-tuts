// Package checker implements the website availability endpoint.
package checker

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

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

// Handler accepts its dependencies as arguments. This makes the handler
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
		// An unbuffered channel passes each result directly to the receiver.
		resultsCh := make(chan indexedResult)
		type job struct {
			index  int
			target string
		}
		jobs := make(chan job)
		// Limit concurrency so a large input file does not start thousands of
		// simultaneous requests. Each worker handles jobs until jobs closes.
		const workerCount = 10
		// Each worker sends one empty value when it exits. The buffer lets
		// every worker signal completion without waiting for the receiver.
		workerDone := make(chan struct{}, workerCount)
		for range workerCount {
			go func() {
				defer func() { workerDone <- struct{}{} }()
				for j := range jobs {
					siteResult := checkSite(r.Context(), client, j.target)
					// If the caller leaves, do not get stuck trying to send
					// a result that nobody is receiving.
					select {
					case resultsCh <- indexedResult{index: j.index, result: siteResult}:
					case <-r.Context().Done():
						return
					}
				}
			}()
		}
		go func() {
			defer close(jobs) // Only the sender closes the jobs channel.
			for i, target := range targets {
				select {
				case jobs <- job{index: i, target: target}:
				case <-r.Context().Done():
					return
				}
			}
		}()
		go func() {
			// Receive one completion signal from each worker before closing
			// resultsCh; no worker can send another result after signaling.
			for range workerCount {
				<-workerDone
			}
			// Close only after every sender has stopped, so sending on a
			// closed channel is impossible. This signals end of results.
			close(resultsCh)
		}()
		// Range receives until resultsCh is closed and drained. It does not
		// need to know how many results will arrive.
		for received := range resultsCh {
			results[received.index] = received.result
		}

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

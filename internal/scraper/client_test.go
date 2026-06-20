package scraper

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_FetchJobs_RequestStructure(t *testing.T) {
	// Setup a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set a mock CSRF cookie on GET or any request if missing
		http.SetCookie(w, &http.Cookie{
			Name:  "CALYPSO_CSRF_TOKEN",
			Value: "mock-token",
		})

		// Check headers for POST requests
		if r.Method == "POST" {
			expectedHeaders := map[string]string{
				"Content-Type":         "application/json",
				"Accept":               "application/json, text/plain, */*",
				"Accept-Language":      "en-US",
				"User-Agent":           "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"X-Calypso-Csrf-Token": "mock-token",
			}

			// First request won't have the token because it hasn't received the cookie yet.
			// But since we are testing stateful behavior, we should probably do a GET first.
			// In our current implementation, we just set it if we have it.
			if r.Header.Get("X-Calypso-Csrf-Token") != "" {
				for k, v := range expectedHeaders {
					if r.Header.Get(k) != v {
						t.Errorf("Expected header %s: %s, got: %s", k, v, r.Header.Get(k))
					}
				}
			}

			// Check body for appliedFacets
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("Failed to read request body: %v", err)
			}

			// Verify appliedFacets is {} and not null by checking the raw JSON
			bodyStr := string(body)
			if bodyStr == "" {
				t.Error("Request body is empty")
			}

			var raw map[string]interface{}
			if err := json.Unmarshal(body, &raw); err != nil {
				t.Fatalf("Failed to unmarshal request body: %v", err)
			}

			if facets, ok := raw["appliedFacets"]; !ok {
				t.Error("appliedFacets missing from request body")
			} else if facets == nil {
				t.Error("appliedFacets is null in request body, expected {}")
			}
		}

		// Return a valid mock response
		resp := WorkdayResponse{
			JobPostings: []JobListing{},
			Total:       0,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client and override URL
	client := NewClient()

	// First call to get the cookie
	client.fetchJobsAt(server.URL)

	// Second call should have the X-Calypso-Csrf-Token header
	_, err := client.fetchJobsAt(server.URL)
	if err != nil {
		t.Fatalf("fetchJobsAt failed: %v", err)
	}
}

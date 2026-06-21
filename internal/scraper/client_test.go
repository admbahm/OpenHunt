package scraper

import (
	"encoding/json"
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

		// Return a valid mock response including facets for the resolution pass
		if r.Method == "POST" {
			// Check headers
			expectedHeaders := map[string]string{
				"Content-Type":         "application/json",
				"Accept":               "application/json",
				"User-Agent":           "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"X-Calypso-Csrf-Token": "mock-token",
			}
			// Note: fetchJobsAt uses a slightly different Accept header than resolveFacetID
			// so we relax this a bit or check specifically.
			for k, v := range expectedHeaders {
				if k == "Accept" {
					continue // Accept differs slightly
				}
				if r.Header.Get(k) != v {
					t.Errorf("Expected header %s: %s, got: %s", k, v, r.Header.Get(k))
				}
			}

			// For simplicity in test, return facets that match our expectations
			facetResp := struct {
				Facets []struct {
					FacetParameter string `json:"facetParameter"`
					Values         []struct {
						Descriptor string `json:"descriptor"`
						ID         string `json:"id"`
					} `json:"values"`
				} `json:"facets"`
				JobPostings []JobListing `json:"jobPostings"`
				Total       int          `json:"total"`
			}{
				Facets: []struct {
					FacetParameter string `json:"facetParameter"`
					Values         []struct {
						Descriptor string `json:"descriptor"`
						ID         string `json:"id"`
					} `json:"values"`
				}{
					{
						FacetParameter: "jobFamilyGroup",
						Values: []struct {
							Descriptor string `json:"descriptor"`
							ID         string `json:"id"`
						}{
							{Descriptor: "Engineering", ID: "engineering-id"},
						},
					},
					{
						FacetParameter: "locations",
						Values: []struct {
							Descriptor string `json:"descriptor"`
							ID         string `json:"id"`
						}{
							{Descriptor: "United States of America", ID: "usa-id"},
							{Descriptor: "San Diego, California", ID: "sd-id"},
						},
					},
				},
				JobPostings: []JobListing{},
				Total:       0,
			}
			json.NewEncoder(w).Encode(facetResp)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client
	client := NewWorkdayScraper(nil)

	target := TargetCompany{
		Name:     "TestCompany",
		Tenant:   "testtenant",
		Site:     "testsite",
		BaseURL:  server.URL,
		Platform: "workday",
		Category: "Engineering",
		Country:  "United States of America",
		Location: "San Diego, California",
	}

	_, err := client.FetchJobs(target)
	if err != nil {
		t.Fatalf("FetchJobs failed: %v", err)
	}
}

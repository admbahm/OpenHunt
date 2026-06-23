package scraper

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
				"Accept-Language":      "en-US",
				"User-Agent":           "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"X-Calypso-Csrf-Token": "mock-token",
				"X-Requested-With":     "XMLHttpRequest",
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

func TestBuildWorkdayDetailURL(t *testing.T) {
	tests := []struct {
		name         string
		jobsEndpoint string
		externalPath string
		want         string
	}{
		{
			name:         "external path already includes job prefix",
			jobsEndpoint: "https://example.com/wday/cxs/acme/careers/jobs",
			externalPath: "/job/California/Engineer_JR123",
			want:         "https://example.com/wday/cxs/acme/careers/job/California/Engineer_JR123",
		},
		{
			name:         "external path without job prefix",
			jobsEndpoint: "https://example.com/wday/cxs/acme/careers/jobs",
			externalPath: "California/Engineer_JR123",
			want:         "https://example.com/wday/cxs/acme/careers/job/California/Engineer_JR123",
		},
		{
			name:         "absolute detail URL",
			jobsEndpoint: "https://example.com/wday/cxs/acme/careers/jobs",
			externalPath: "https://jobs.example.net/wday/cxs/acme/careers/job/Engineer_JR123",
			want:         "https://jobs.example.net/wday/cxs/acme/careers/job/Engineer_JR123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildWorkdayDetailURL(tt.jobsEndpoint, tt.externalPath)
			if err != nil {
				t.Fatalf("buildWorkdayDetailURL returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("detail URL = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFetchJobDescriptionUsesNormalizedDetailPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wday/cxs/acme/careers/job/California/Engineer_JR123" {
			t.Fatalf("request path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"jobPostingInfo": map[string]string{
				"jobDescription": "Build reliable systems.",
			},
		})
	}))
	defer server.Close()

	client := NewWorkdayScraper(server.Client())
	description, err := client.fetchJobDescription(
		server.URL+"/wday/cxs/acme/careers/jobs",
		"/job/California/Engineer_JR123",
	)
	if err != nil {
		t.Fatalf("fetchJobDescription returned error: %v", err)
	}
	if description != "Build reliable systems." {
		t.Fatalf("description = %q", description)
	}
}

func TestFetchJobsRemapsNestedLocationFacetsInSearchPayload(t *testing.T) {
	var searchPayload WorkdayRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.SetCookie(w, &http.Cookie{
				Name:  "CALYPSO_CSRF_TOKEN",
				Value: "mock-token",
			})
			w.WriteHeader(http.StatusOK)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}

		var request WorkdayRequest
		if err := json.Unmarshal(body, &request); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if len(request.AppliedFacets) == 0 {
			// Workday groups location values beneath locationMainGroup in the
			// discovery response, but rejects that grouping key in searches.
			_, _ = w.Write([]byte(`{
				"facets": [
					{
						"facetParameter": "jobFamilyGroup",
						"values": [
							{"descriptor": "Engineering", "id": "engineering-id"}
						]
					},
					{
						"facetParameter": "locationMainGroup",
						"values": [
							{"descriptor": "United States of America", "id": "usa-id"}
						]
					},
					{
						"facetParameter": "locations",
						"values": [
							{"descriptor": "San Diego, California", "id": "sd-id"}
						]
					}
				],
				"jobPostings": [],
				"total": 0
			}`))
			return
		}

		searchPayload = request
		_, _ = w.Write([]byte(`{"jobPostings":[],"total":0}`))
	}))
	defer server.Close()

	client := NewWorkdayScraper(server.Client())
	_, err := client.FetchJobs(TargetCompany{
		Name:     "Acme",
		Tenant:   "acme",
		Site:     "careers",
		BaseURL:  server.URL,
		Platform: "workday",
		Category: "Engineering",
		Country:  "United States of America",
		Location: "San Diego, California",
	})
	if err != nil {
		t.Fatalf("FetchJobs returned error: %v", err)
	}

	if _, exists := searchPayload.AppliedFacets["locationMainGroup"]; exists {
		t.Fatalf("search payload contains unsupported locationMainGroup: %#v", searchPayload.AppliedFacets)
	}
	assertFacetIDs(t, searchPayload.AppliedFacets, "jobFamilyGroup", []string{"engineering-id"})
	assertFacetIDs(t, searchPayload.AppliedFacets, "locationCountry", []string{"usa-id"})
	assertFacetIDs(t, searchPayload.AppliedFacets, "locations", []string{"sd-id"})
}

func TestFetchJobsUsesNestedFacetParameterForWorkdayLocationGroups(t *testing.T) {
	var searchPayload WorkdayRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}

		var request WorkdayRequest
		if err := json.Unmarshal(body, &request); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if len(request.AppliedFacets) == 0 {
			_, _ = w.Write([]byte(`{
				"facets": [
					{
						"facetParameter": "jobFamilyGroup",
						"values": [
							{"descriptor": "Engineering", "id": "engineering-id"}
						]
					},
					{
						"facetParameter": "locationMainGroup",
						"values": [
							{
								"facetParameter": "locationHierarchy1",
								"descriptor": "Locations",
								"values": [
									{"descriptor": "United States", "id": "usa-id"}
								]
							},
							{
								"facetParameter": "locations",
								"descriptor": "Sites",
								"values": [
									{"descriptor": "US, CA, Santa Clara", "id": "santa-clara-id"}
								]
							}
						]
					}
				],
				"jobPostings": [],
				"total": 0
			}`))
			return
		}

		searchPayload = request
		_, _ = w.Write([]byte(`{"jobPostings":[],"total":0}`))
	}))
	defer server.Close()

	client := NewWorkdayScraper(server.Client())
	_, err := client.FetchJobs(TargetCompany{
		Name:     "NVIDIA",
		Tenant:   "nvidia",
		Site:     "NVIDIAExternalCareerSite",
		BaseURL:  server.URL,
		Platform: "workday",
		Category: "Engineering",
		Country:  "United States of America",
		Location: "San Diego, California",
	})
	if err != nil {
		t.Fatalf("FetchJobs returned error: %v", err)
	}

	assertFacetIDs(t, searchPayload.AppliedFacets, "jobFamilyGroup", []string{"engineering-id"})
	assertFacetIDs(t, searchPayload.AppliedFacets, "locationHierarchy1", []string{"usa-id"})
	if _, exists := searchPayload.AppliedFacets["locations"]; exists {
		t.Fatalf("country ID should not be submitted under locations: %#v", searchPayload.AppliedFacets)
	}
}

func TestFetchJobsRetriesTransientWorkdayFailure(t *testing.T) {
	searchAttempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			return
		}

		searchAttempts++
		if searchAttempts == 1 {
			http.Error(w, `{"httpStatus":502}`, http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"jobPostings": [{
				"bulletinNumber": "JR123",
				"title": "Engineer",
				"locationsText": "San Diego, California",
				"externalPath": "/job/California/Engineer_JR123"
			}],
			"total": 1
		}`))
	}))
	defer server.Close()

	client := NewWorkdayScraper(server.Client())
	client.retryDelay = func(int) time.Duration { return 0 }
	client.sleep = func(time.Duration) {}

	jobs, err := client.FetchJobs(TargetCompany{
		Name:     "Acme",
		Tenant:   "acme",
		Site:     "careers",
		BaseURL:  server.URL,
		Platform: "workday",
	})
	if err != nil {
		t.Fatalf("FetchJobs returned error: %v", err)
	}
	if searchAttempts != 2 {
		t.Fatalf("search attempts = %d, want 2", searchAttempts)
	}
	if len(jobs) != 1 || jobs[0].JobID != "JR123" {
		t.Fatalf("jobs = %#v, want recovered JR123 listing", jobs)
	}
	wantURL := server.URL + "/wday/cxs/acme/careers/job/California/Engineer_JR123"
	if jobs[0].ExternalPath != wantURL {
		t.Fatalf("ExternalPath = %q, want %q", jobs[0].ExternalPath, wantURL)
	}
}

func TestFetchJobsDoesNotRetryPermanentWorkdayFailure(t *testing.T) {
	searchAttempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			return
		}
		searchAttempts++
		http.Error(w, `{"httpStatus":400}`, http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewWorkdayScraper(server.Client())
	client.retryDelay = func(int) time.Duration { return 0 }
	client.sleep = func(time.Duration) {}

	_, err := client.FetchJobs(TargetCompany{
		Name:     "Acme",
		Tenant:   "acme",
		Site:     "careers",
		BaseURL:  server.URL,
		Platform: "workday",
	})
	if err == nil {
		t.Fatal("FetchJobs returned nil error for HTTP 400")
	}
	if searchAttempts != 1 {
		t.Fatalf("search attempts = %d, want 1", searchAttempts)
	}
}

func assertFacetIDs(t *testing.T, facets map[string][]string, key string, want []string) {
	t.Helper()
	got, exists := facets[key]
	if !exists {
		t.Fatalf("missing applied facet %q in %#v", key, facets)
	}
	if len(got) != len(want) {
		t.Fatalf("facet %q = %#v, want %#v", key, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("facet %q = %#v, want %#v", key, got, want)
		}
	}
}

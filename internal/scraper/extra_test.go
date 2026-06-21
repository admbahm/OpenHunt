package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewScraperFactory(t *testing.T) {
	tests := []struct {
		platform string
		wantErr  bool
	}{
		{"workday", false},
		{"greenhouse", false},
		{"invalid", true},
	}

	for _, tt := range tests {
		s, err := NewScraperFactory(tt.platform, nil)
		if (err != nil) != tt.wantErr {
			t.Errorf("NewScraperFactory(%s) error = %v, wantErr %v", tt.platform, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && s == nil {
			t.Errorf("NewScraperFactory(%s) returned nil scraper", tt.platform)
		}
	}
}

func TestGreenhouseScraper_FetchJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Received request: %s", r.URL.String())

		if !strings.HasPrefix(r.URL.Path, "/v1/boards/test-tenant/jobs") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := GreenhouseResponse{
			Jobs: []struct {
				ID       int64  `json:"id"`
				Title    string `json:"title"`
				Location struct {
					Name string `json:"name"`
				} `json:"location"`
				AbsoluteURL string `json:"absolute_url"`
				Departments []struct {
					Name string `json:"name"`
				} `json:"departments"`
			}{
				{
					ID:    123,
					Title: "Test Job",
					Location: struct {
						Name string `json:"name"`
					}{Name: "Test Location"},
					AbsoluteURL: "http://example.com/job/123",
					Departments: []struct {
						Name string `json:"name"`
					}{
						{Name: "Engineering"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	scraper := &GreenhouseScraper{
		Client:  server.Client(),
		BaseURL: server.URL,
	}
	target := TargetCompany{
		Tenant: "test-tenant",
	}

	listings, err := scraper.FetchJobs(target)
	if err != nil {
		t.Fatalf("FetchJobs failed: %v", err)
	}

	if len(listings) != 1 {
		t.Errorf("Expected 1 listing, got %d", len(listings))
	}

	if listings[0].JobID != "gh-123" {
		t.Errorf("Expected JobID gh-123, got %s", listings[0].JobID)
	}
}

func TestGreenhouseScraper_FetchJobs_LooseFiltering(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := GreenhouseResponse{
			Jobs: []struct {
				ID       int64  `json:"id"`
				Title    string `json:"title"`
				Location struct {
					Name string `json:"name"`
				} `json:"location"`
				AbsoluteURL string `json:"absolute_url"`
				Departments []struct {
					Name string `json:"name"`
				} `json:"departments"`
			}{
				{
					ID:    1,
					Title: "Stripe Eng Job 1",
					Location: struct {
						Name string `json:"name"`
					}{Name: "Remote, US"},
					AbsoluteURL: "http://example.com/job/1",
					Departments: []struct {
						Name string `json:"name"`
					}{
						{Name: "Software Engineering"},
					},
				},
				{
					ID:    2,
					Title: "Stripe Eng Job 2",
					Location: struct {
						Name string `json:"name"`
					}{Name: "San Francisco, CA"},
					AbsoluteURL: "http://example.com/job/2",
					Departments: []struct {
						Name string `json:"name"`
					}{
						{Name: "Software Engineering"},
					},
				},
				{
					ID:    3,
					Title: "Stripe Sales Job",
					Location: struct {
						Name string `json:"name"`
					}{Name: "Remote"},
					AbsoluteURL: "http://example.com/job/3",
					Departments: []struct {
						Name string `json:"name"`
					}{
						{Name: "Sales & Marketing"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	scraper := &GreenhouseScraper{
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	// Case 1: Target Category "1. Engineering" and Location "6. Remote" (tests prefix stripping and lowercase matching)
	// Should match Job 1 ("Software Engineering", "Remote, US")
	// Should skip Job 2 (mismatch location: "San Francisco, CA")
	// Should skip Job 3 (mismatch category: "Sales & Marketing")
	target := TargetCompany{
		Tenant:   "test-tenant",
		Category: "1. Engineering",
		Location: "6. Remote",
	}

	listings, err := scraper.FetchJobs(target)
	if err != nil {
		t.Fatalf("FetchJobs failed: %v", err)
	}

	if len(listings) != 1 {
		t.Errorf("Expected 1 listing, got %d", len(listings))
	} else {
		if listings[0].JobID != "gh-1" {
			t.Errorf("Expected JobID gh-1, got %s", listings[0].JobID)
		}
	}
}


func TestWorkdayScraper_ResolveFacetID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock response for facet resolution
		resp := struct {
			Facets []struct {
				FacetParameter string `json:"facetParameter"`
				Values         []struct {
					Descriptor string `json:"descriptor"`
					ID         string `json:"id"`
					Values     []struct {
						Descriptor string `json:"descriptor"`
						ID         string `json:"id"`
					} `json:"values"`
				} `json:"values"`
			} `json:"facets"`
		}{
			Facets: []struct {
				FacetParameter string `json:"facetParameter"`
				Values         []struct {
					Descriptor string `json:"descriptor"`
					ID         string `json:"id"`
					Values     []struct {
						Descriptor string `json:"descriptor"`
						ID         string `json:"id"`
					} `json:"values"`
				} `json:"values"`
			}{
				{
					FacetParameter: "jobFamilyGroup",
					Values: []struct {
						Descriptor string `json:"descriptor"`
						ID         string `json:"id"`
						Values     []struct {
							Descriptor string `json:"descriptor"`
							ID         string `json:"id"`
						} `json:"values"`
					}{
						{
							Descriptor: "Engineering & Development",
							ID:         "eng-dev-id",
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	scraper := NewWorkdayScraper(server.Client())

	// Test case-insensitive partial match
	id, param, err := scraper.resolveFacetID(server.URL, "category", "engineering")
	if err != nil {
		t.Fatalf("resolveFacetID failed: %v", err)
	}
	if id != "eng-dev-id" {
		t.Errorf("Expected id eng-dev-id, got %s", id)
	}
	if param != "jobFamilyGroup" {
		t.Errorf("Expected param jobFamilyGroup, got %s", param)
	}
}

func TestScraper_Run(t *testing.T) {
	// Mock server for Greenhouse
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"jobs": [{"id": 1, "title": "GH Job", "location": {"name": "Remote"}, "absolute_url": "http://gh.com/1"}]}`)
	}))
	defer ghServer.Close()

	// We can't easily mock Workday because it uses a different URL structure
	// and Scraper uses NewScraperFactory which uses hardcoded URLs.
	// However, we can test with just Greenhouse if we modify the target to look like a greenhouse target
	// AND we need to point GreenhouseScraper to our mock server.
	// This is hard because NewScraperFactory instantiates GreenhouseScraper with default BaseURL.

	// Let's test the Scraper logic by providing a list of companies.
	s := NewScraper(2)
	companies := []TargetCompany{
		{Name: "Invalid", Platform: "invalid"},
	}

	results := s.Run(companies)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].Error == nil {
		t.Error("Expected error for invalid platform, got nil")
	}
}

func TestWorkdayScraper_ResolveFacetID_LocationMainGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Facets []struct {
				FacetParameter string `json:"facetParameter"`
				Values         []struct {
					Descriptor string `json:"descriptor"`
					ID         string `json:"id"`
					Values     []struct {
						Descriptor string `json:"descriptor"`
						ID         string `json:"id"`
					} `json:"values"`
				} `json:"values"`
			} `json:"facets"`
		}{
			Facets: []struct {
				FacetParameter string `json:"facetParameter"`
				Values         []struct {
					Descriptor string `json:"descriptor"`
					ID         string `json:"id"`
					Values     []struct {
						Descriptor string `json:"descriptor"`
						ID         string `json:"id"`
					} `json:"values"`
				} `json:"values"`
			}{
				{
					FacetParameter: "locationMainGroup",
					Values: []struct {
						Descriptor string `json:"descriptor"`
						ID         string `json:"id"`
						Values     []struct {
							Descriptor string `json:"descriptor"`
							ID         string `json:"id"`
						} `json:"values"`
					}{
						{
							Descriptor: "Country",
							ID:         "",
							Values: []struct {
								Descriptor string `json:"descriptor"`
								ID         string `json:"id"`
							}{
								{Descriptor: "United States of America", ID: "usa-id"},
							},
						},
						{
							Descriptor: "Locations",
							ID:         "",
							Values: []struct {
								Descriptor string `json:"descriptor"`
								ID         string `json:"id"`
							}{
								{Descriptor: "San Diego, California", ID: "sd-id"},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	scraper := NewWorkdayScraper(server.Client())

	// Test Country resolution maps locationMainGroup -> locationCountry
	id, param, err := scraper.resolveFacetID(server.URL, "location", "United States of America")
	if err != nil {
		t.Fatalf("resolveFacetID failed: %v", err)
	}
	if id != "usa-id" {
		t.Errorf("Expected id usa-id, got %s", id)
	}
	if param != "locationCountry" {
		t.Errorf("Expected param locationCountry, got %s", param)
	}

	// Test Location resolution maps locationMainGroup -> locations
	id, param, err = scraper.resolveFacetID(server.URL, "location", "San Diego, California")
	if err != nil {
		t.Fatalf("resolveFacetID failed: %v", err)
	}
	if id != "sd-id" {
		t.Errorf("Expected id sd-id, got %s", id)
	}
	if param != "locations" {
		t.Errorf("Expected param locations, got %s", param)
	}
}


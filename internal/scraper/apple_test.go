package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppleScraperFetchJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/en-us/search":
			// 1. GET Landing Page
			if r.Method != "GET" {
				t.Fatalf("landing page: method = %q, want GET", r.Method)
			}
			w.Header().Add("Set-Cookie", "jobs=mock-jobs-cookie")
			w.Header().Add("Set-Cookie", "cs-id=mock-cs-id-cookie")
			w.Header().Add("Set-Cookie", "AWSALBAPP-0=mock-aws-cookie")
			w.WriteHeader(http.StatusOK)

		case "/api/v1/CSRFToken":
			// 2. GET CSRFToken
			if r.Method != "GET" {
				t.Fatalf("CSRFToken: method = %q, want GET", r.Method)
			}
			cookieHeader := r.Header.Get("Cookie")
			if !strings.Contains(cookieHeader, "jobs=mock-jobs-cookie") ||
				!strings.Contains(cookieHeader, "cs-id=mock-cs-id-cookie") ||
				!strings.Contains(cookieHeader, "AWSALBAPP-0=mock-aws-cookie") {
				t.Fatalf("CSRFToken: missing expected cookies in request, got %q", cookieHeader)
			}
			w.Header().Set("X-Apple-CSRF-Token", "mock-csrf-token-val")
			w.Header().Add("Set-Cookie", "jssid=mock-jssid-cookie")
			w.WriteHeader(http.StatusOK)

		case "/api/v1/search":
			// 3. POST search
			if r.Method != "POST" {
				t.Fatalf("search: method = %q, want POST", r.Method)
			}
			csrf := r.Header.Get("X-Apple-CSRF-Token")
			if csrf != "mock-csrf-token-val" {
				t.Fatalf("search: csrf token = %q, want mock-csrf-token-val", csrf)
			}
			cookieHeader := r.Header.Get("Cookie")
			if !strings.Contains(cookieHeader, "jssid=mock-jssid-cookie") {
				t.Fatalf("search: missing jssid cookie in request, got %q", cookieHeader)
			}

			// Parse payload
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode search request payload: %v", err)
			}
			query, _ := payload["query"].(string)
			if query != "Software" {
				t.Fatalf("search payload query = %q, want Software", query)
			}

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"res": {
					"totalRecords": 1,
					"searchResults": [
						{
							"id": "200669114",
							"jobPositionId": "REQ-200669114",
							"jobSummary": "Build great software.",
							"postingTitle": "Software Development Engineer",
							"postingDate": "Jun 26, 2026",
							"transformedPostingTitle": "software-development-engineer",
							"positionId": "200669114",
							"homeOffice": false,
							"team": {
								"teamCode": "SFTWR",
								"teamName": "Software and Services"
							},
							"locations": [
								{
									"name": "Cupertino",
									"city": "Cupertino",
									"stateProvince": "CA",
									"countryName": "United States of America",
									"countryID": "iso-country-USA"
								}
							]
						}
					]
				}
			}`)

		default:
			t.Fatalf("unexpected request path: %q", r.URL.Path)
		}
	}))
	defer server.Close()

	scraper := &AppleScraper{Client: server.Client(), BaseURL: server.URL}
	jobs, err := scraper.FetchJobs(TargetCompany{
		Platform: "apple",
		Category: "Software",
		Country:  "USA",
	})
	if err != nil {
		t.Fatalf("FetchJobs failed: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	job := jobs[0]
	if job.JobID != "apple-200669114" {
		t.Fatalf("JobID = %q, want apple-200669114", job.JobID)
	}
	if job.Title != "Software Development Engineer" {
		t.Fatalf("Title = %q, want Software Development Engineer", job.Title)
	}
	if job.LocationsText != "Cupertino, CA (United States of America)" {
		t.Fatalf("LocationsText = %q, want Cupertino, CA (United States of America)", job.LocationsText)
	}
	if job.ExternalPath != "https://jobs.apple.com/en-us/details/200669114/software-development-engineer" {
		t.Fatalf("ExternalPath = %q", job.ExternalPath)
	}
	if job.Description != "Build great software." {
		t.Fatalf("Description = %q", job.Description)
	}
	if job.PostedOn != "Jun 26, 2026" {
		t.Fatalf("PostedOn = %q", job.PostedOn)
	}
}

func TestAppleScraperStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/en-us/search" {
			w.Header().Add("Set-Cookie", "cs-id=mock-cs-id")
			w.WriteHeader(http.StatusOK)
		} else if r.URL.Path == "/api/v1/CSRFToken" {
			w.Header().Set("X-Apple-CSRF-Token", "mock-token")
			w.WriteHeader(http.StatusOK)
		} else {
			http.Error(w, "internal service error", http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()

	scraper := &AppleScraper{Client: server.Client(), BaseURL: server.URL}
	_, err := scraper.FetchJobs(TargetCompany{
		Platform: "apple",
		Category: "Software",
		Country:  "USA",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "apple search api returned status 503") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestAppleScraperWithLocationFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/en-us/search":
			w.Header().Add("Set-Cookie", "jobs=mock-jobs")
			w.WriteHeader(http.StatusOK)
		case "/api/v1/CSRFToken":
			w.Header().Set("X-Apple-CSRF-Token", "mock-csrf")
			w.WriteHeader(http.StatusOK)
		case "/api/v1/refData/postlocation":
			if r.URL.Query().Get("input") != "San Diego" {
				t.Fatalf("autocomplete: input = %q, want San Diego", r.URL.Query().Get("input"))
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"res":[{"id":"postLocation-SDO","name":"San Diego, CA","city":"San Diego","stateProvince":"California"}]}`)
		case "/api/v1/search":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode search payload: %v", err)
			}
			filters, _ := payload["filters"].(map[string]interface{})
			locations, _ := filters["locations"].([]interface{})
			if len(locations) != 1 || locations[0].(string) != "postLocation-SDO" {
				t.Fatalf("search payload locations = %v, want [postLocation-SDO]", locations)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"res": {
					"totalRecords": 1,
					"searchResults": [
						{
							"id": "200669114",
							"postingTitle": "Software Engineer",
							"postingDate": "Jun 28, 2026",
							"positionId": "200669114",
							"locations": [
								{
									"name": "San Diego",
									"city": "San Diego",
									"stateProvince": "CA",
									"countryName": "United States of America",
									"countryID": "iso-country-USA"
								}
							]
						}
					]
				}
			}`)
		default:
			t.Fatalf("unexpected request: %q", r.URL.Path)
		}
	}))
	defer server.Close()

	scraper := &AppleScraper{Client: server.Client(), BaseURL: server.URL}
	jobs, err := scraper.FetchJobs(TargetCompany{
		Platform: "apple",
		Category: "Software",
		Location: "San Diego, California",
		Country:  "USA",
	})
	if err != nil {
		t.Fatalf("FetchJobs failed: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].LocationsText != "San Diego, CA (United States of America)" {
		t.Fatalf("LocationsText = %q", jobs[0].LocationsText)
	}
}

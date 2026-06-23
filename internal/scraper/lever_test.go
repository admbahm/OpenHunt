package scraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLeverScraperFetchJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/postings/acme" {
			t.Fatalf("path = %q, want /v0/postings/acme", r.URL.Path)
		}
		if r.URL.Query().Get("mode") != "json" {
			t.Fatalf("mode query = %q, want json", r.URL.Query().Get("mode"))
		}
		if got := r.Header.Get("User-Agent"); got != "openHunt/2.0" {
			t.Fatalf("User-Agent = %q, want openHunt/2.0", got)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{
				"id": "posting-1",
				"title": "Software Engineer",
				"workplaceType": "remote",
				"hostedUrl": "https://jobs.lever.co/acme/posting-1",
				"description": "Build reliable systems."
			}
		]`)
	}))
	defer server.Close()

	scraper := &LeverScraper{Client: server.Client(), BaseURL: server.URL}
	jobs, err := scraper.FetchJobs(TargetCompany{Tenant: "acme"})
	if err != nil {
		t.Fatalf("FetchJobs returned error: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d: %#v", len(jobs), jobs)
	}
	job := jobs[0]
	if job.JobID != "lever-posting-1" {
		t.Fatalf("JobID = %q, want lever-posting-1", job.JobID)
	}
	if job.Title != "Software Engineer" {
		t.Fatalf("Title = %q, want Software Engineer", job.Title)
	}
	if job.LocationsText != "remote" {
		t.Fatalf("LocationsText = %q, want remote", job.LocationsText)
	}
	if job.ExternalPath != "https://jobs.lever.co/acme/posting-1" {
		t.Fatalf("ExternalPath = %q", job.ExternalPath)
	}
	if job.Description != "Build reliable systems." {
		t.Fatalf("Description = %q", job.Description)
	}
}

func TestLeverScraperReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	scraper := &LeverScraper{Client: server.Client(), BaseURL: server.URL}
	_, err := scraper.FetchJobs(TargetCompany{Tenant: "missing"})
	if err == nil {
		t.Fatal("FetchJobs returned nil error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "lever api returned status: 404") {
		t.Fatalf("unexpected error: %v", err)
	}
}

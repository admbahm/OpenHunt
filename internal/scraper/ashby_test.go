package scraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAshbyScraperFetchesPublicBoardAndDescriptions(t *testing.T) {
	var detailRequested bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/linear":
			fmt.Fprint(w, ashbyHTML(`{
				"jobBoard": {
					"jobPostings": [
						{
							"id": "engineering-job",
							"title": "Software Engineer",
							"departmentName": "Engineering",
							"teamName": "Product Engineering",
							"locationName": "United States",
							"workplaceType": "Remote",
							"publishedDate": "2026-06-23"
						},
						{
							"id": "sales-job",
							"title": "Account Executive",
							"departmentName": "Sales",
							"locationName": "United States",
							"workplaceType": "Remote"
						}
					]
				},
				"posting": null
			}`))
		case "/linear/engineering-job":
			detailRequested = true
			fmt.Fprint(w, ashbyHTML(`{
				"jobBoard": {"jobPostings": []},
				"posting": {
					"id": "engineering-job",
					"descriptionHtml": "<p>Build product systems &amp; developer workflows.</p>"
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	scraper := &AshbyScraper{Client: server.Client(), BaseURL: server.URL}
	jobs, err := scraper.FetchJobs(TargetCompany{
		Tenant:   "linear",
		Category: "Engineering",
		Location: "Remote",
	})
	if err != nil {
		t.Fatalf("FetchJobs returned error: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 filtered job, got %d: %#v", len(jobs), jobs)
	}
	if !detailRequested {
		t.Fatal("expected scraper to fetch matched posting detail page")
	}

	job := jobs[0]
	if job.JobID != "ashby-engineering-job" {
		t.Fatalf("expected prefixed ashby job id, got %q", job.JobID)
	}
	if job.Title != "Software Engineer" {
		t.Fatalf("expected title from public board, got %q", job.Title)
	}
	if job.LocationsText != "United States (Remote)" {
		t.Fatalf("expected combined location, got %q", job.LocationsText)
	}
	if job.ExternalPath != server.URL+"/linear/engineering-job" {
		t.Fatalf("expected public posting URL, got %q", job.ExternalPath)
	}
	if job.Description != "Build product systems & developer workflows." {
		t.Fatalf("expected text description from detail page, got %q", job.Description)
	}
}

func TestExtractAshbyAppDataRequiresEmbeddedData(t *testing.T) {
	if _, err := extractAshbyAppData(httptest.NewRecorder().Body); err == nil {
		t.Fatal("expected missing app data to return an error")
	}
}

func TestExtractAshbyAppDataAllowsAdditionalScriptAfterAssignment(t *testing.T) {
	page := `<script>
		window.__appData = {
			"jobBoard": {
				"jobPostings": [
					{
						"id": "posting-with-braces",
						"title": "Engineer",
						"descriptionHtml": "<p>Uses {braces} in text</p>"
					}
				]
			},
			"posting": null
		};
		window.addEventListener("load", function () {
			console.log("extra ashby boot code");
		});
	</script>`

	ashbyResp, err := extractAshbyAppData(strings.NewReader(page))
	if err != nil {
		t.Fatalf("extractAshbyAppData returned error: %v", err)
	}

	if got := ashbyResp.JobBoard.JobPostings[0].ID; got != "posting-with-braces" {
		t.Fatalf("expected posting id from embedded app data, got %q", got)
	}
}

func ashbyHTML(appData string) string {
	return `<html><body><script>window.__appData = ` + appData + `;</script></body></html>`
}

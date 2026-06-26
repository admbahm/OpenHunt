package validation

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/openhunt/openhunt/internal/scraper"
)

type fakeScraper struct {
	jobs []scraper.JobListing
	err  error
}

func (f fakeScraper) FetchJobs(scraper.TargetCompany) ([]scraper.JobListing, error) {
	return f.jobs, f.err
}

func TestValidateReportsMetadataErrors(t *testing.T) {
	result := Validator{}.Validate(scraper.TargetCompany{
		Name:     "Acme",
		Platform: "workday",
		Tenant:   "acme",
	})

	if result.OK {
		t.Fatal("expected validation to fail")
	}
	assertContains(t, result.Errors, "site is required for workday targets")
	assertContains(t, result.Errors, "base_url is required for workday targets")
}

func TestValidateReportsUnsupportedPlatform(t *testing.T) {
	result := Validator{}.Validate(scraper.TargetCompany{
		Name:     "Acme",
		Platform: "custom",
		Tenant:   "acme",
	})

	if result.OK {
		t.Fatal("expected validation to fail")
	}
	assertContains(t, result.Errors, "unsupported platform: custom")
}

func TestValidateReportsScrapeError(t *testing.T) {
	validator := Validator{
		Factory: func(string, *http.Client) (scraper.JobScraper, error) {
			return fakeScraper{err: errors.New("temporary failure")}, nil
		},
	}

	result := validator.Validate(scraper.TargetCompany{
		Name:     "Acme",
		Platform: "greenhouse",
		Tenant:   "acme",
	})

	if result.OK {
		t.Fatal("expected validation to fail")
	}
	assertContains(t, result.Errors, "temporary failure")
}

func TestValidateTreatsZeroJobsAsFailure(t *testing.T) {
	validator := Validator{
		Factory: func(string, *http.Client) (scraper.JobScraper, error) {
			return fakeScraper{}, nil
		},
	}

	result := validator.Validate(scraper.TargetCompany{
		Name:     "Acme",
		Platform: "ashby",
		Tenant:   "acme",
	})

	if result.OK {
		t.Fatal("expected validation to fail")
	}
	assertContains(t, result.Errors, "returned zero jobs")
}

func TestValidateSuccessWithWarningsAndSamples(t *testing.T) {
	validator := Validator{
		Factory: func(string, *http.Client) (scraper.JobScraper, error) {
			return fakeScraper{jobs: []scraper.JobListing{
				{
					JobID:         "job-1",
					Title:         "Firmware Engineer",
					LocationsText: "San Diego, California",
					ExternalPath:  "https://example.com/job-1",
					Description:   "Build embedded systems.",
				},
				{
					JobID: "job-2",
				},
			}}, nil
		},
	}

	result := validator.Validate(scraper.TargetCompany{
		Name:     "Acme",
		Platform: "lever",
		Tenant:   "acme",
	})

	if !result.OK {
		t.Fatalf("expected validation to pass, errors: %v", result.Errors)
	}
	if result.JobCount != 2 {
		t.Fatalf("JobCount = %d, want 2", result.JobCount)
	}
	if result.DescriptionCount != 1 {
		t.Fatalf("DescriptionCount = %d, want 1", result.DescriptionCount)
	}
	if len(result.SampleJobs) != 2 {
		t.Fatalf("SampleJobs length = %d, want 2", len(result.SampleJobs))
	}
	assertContains(t, result.Warnings, "job 2 has empty title")
	assertContains(t, result.Warnings, "job 2 has empty apply URL")
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if strings.Contains(value, want) {
			return
		}
	}
	t.Fatalf("%q not found in %v", want, values)
}

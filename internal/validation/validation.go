package validation

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/openhunt/openhunt/internal/scraper"
)

// ScraperFactory creates a scraper for a platform.
type ScraperFactory func(platform string, httpClient *http.Client) (scraper.JobScraper, error)

// Validator checks whether configured ATS targets can be scraped.
type Validator struct {
	HTTPClient *http.Client
	Factory    ScraperFactory
}

// Result contains the validation outcome for one target company.
type Result struct {
	Company          string                `json:"company"`
	Platform         string                `json:"platform"`
	Tenant           string                `json:"tenant"`
	Site             string                `json:"site,omitempty"`
	BaseURL          string                `json:"base_url,omitempty"`
	OK               bool                  `json:"ok"`
	JobCount         int                   `json:"job_count"`
	DescriptionCount int                   `json:"description_count"`
	Duration         time.Duration         `json:"duration"`
	Errors           []string              `json:"errors,omitempty"`
	Warnings         []string              `json:"warnings,omitempty"`
	SampleJobs       []SampleJob           `json:"sample_jobs,omitempty"`
	Target           scraper.TargetCompany `json:"-"`
}

// SampleJob is a compact representation of a returned posting.
type SampleJob struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Location string `json:"location"`
	URL      string `json:"url"`
}

// ValidateAll validates targets sequentially.
func (v Validator) ValidateAll(targets []scraper.TargetCompany) []Result {
	results := make([]Result, 0, len(targets))
	for _, target := range targets {
		results = append(results, v.Validate(target))
	}
	return results
}

// Validate checks target metadata and performs a real scrape request.
func (v Validator) Validate(target scraper.TargetCompany) Result {
	start := time.Now()
	result := Result{
		Company:  target.Name,
		Platform: target.Platform,
		Tenant:   target.Tenant,
		Site:     target.Site,
		BaseURL:  target.BaseURL,
		Target:   target,
	}
	finish := func() Result {
		result.Duration = time.Since(start)
		return result
	}

	result.Errors = append(result.Errors, metadataErrors(target)...)
	if len(result.Errors) > 0 {
		return finish()
	}

	factory := v.Factory
	if factory == nil {
		factory = scraper.NewScraperFactory
	}

	jobScraper, err := factory(target.Platform, v.HTTPClient)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return finish()
	}

	jobs, err := jobScraper.FetchJobs(target)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return finish()
	}

	result.JobCount = len(jobs)
	if result.JobCount == 0 {
		result.Errors = append(result.Errors, "scrape succeeded but returned zero jobs; target may be stale, misidentified, or filtered too narrowly")
	}

	for i, job := range jobs {
		if strings.TrimSpace(job.JobID) == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("job %d has empty id", i+1))
		}
		if strings.TrimSpace(job.Title) == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("job %d has empty title", i+1))
		}
		if strings.TrimSpace(job.ExternalPath) == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("job %d has empty apply URL", i+1))
		}
		if strings.TrimSpace(job.Description) != "" {
			result.DescriptionCount++
		}
		if len(result.SampleJobs) < 3 {
			result.SampleJobs = append(result.SampleJobs, SampleJob{
				ID:       job.JobID,
				Title:    job.Title,
				Location: job.LocationsText,
				URL:      job.ExternalPath,
			})
		}
	}

	if result.JobCount > 0 && result.DescriptionCount == 0 {
		result.Warnings = append(result.Warnings, "no descriptions were extracted")
	}

	result.OK = len(result.Errors) == 0
	return finish()
}

func metadataErrors(target scraper.TargetCompany) []string {
	var errs []string
	if strings.TrimSpace(target.Name) == "" {
		errs = append(errs, "name is required")
	}
	if strings.TrimSpace(target.Platform) == "" {
		errs = append(errs, "platform is required")
	}
	if strings.TrimSpace(target.Tenant) == "" {
		errs = append(errs, "tenant is required")
	}

	switch target.Platform {
	case "workday":
		if strings.TrimSpace(target.Site) == "" {
			errs = append(errs, "site is required for workday targets")
		}
		if strings.TrimSpace(target.BaseURL) == "" {
			errs = append(errs, "base_url is required for workday targets")
		}
	case "greenhouse", "lever", "ashby", "apple":
		// Supported platform.
	case "":
		// Already reported above.
	default:
		errs = append(errs, fmt.Sprintf("unsupported platform: %s", target.Platform))
	}

	return errs
}

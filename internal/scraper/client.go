package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client handles communication with Workday CXS endpoints.
type Client struct {
	httpClient *http.Client
	userAgent  string
}

// NewClient initializes a new Client with default settings.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

// FetchJobs retrieves job listings for a given target company.
func (c *Client) FetchJobs(target TargetCompany) ([]JobListing, error) {
	url := fmt.Sprintf("https://%s.wd3.myworkdayjobs.com/wday/cxs/%s/%s/jobs", target.Tenant, target.Tenant, target.Site)

	reqPayload := WorkdayRequest{
		AppliedFacets: make(map[string][]string),
		Limit:         20,
		Offset:        0,
		SearchText:    "",
	}

	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	var workdayResp WorkdayResponse
	if err := json.NewDecoder(resp.Body).Decode(&workdayResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return workdayResp.JobPostings, nil
}

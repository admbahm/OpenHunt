package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

// WorkdayScraper handles communication with Workday CXS endpoints.
type WorkdayScraper struct {
	httpClient *http.Client
	userAgent  string
}

// NewWorkdayScraper initializes a new WorkdayScraper with default settings.
func NewWorkdayScraper(client *http.Client) *WorkdayScraper {
	if client == nil {
		jar, _ := cookiejar.New(nil)
		client = &http.Client{
			Timeout: 15 * time.Second,
			Jar:     jar,
		}
	}
	return &WorkdayScraper{
		httpClient: client,
		userAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

// FetchJobs retrieves job listings for a given target company.
func (c *WorkdayScraper) FetchJobs(target TargetCompany) ([]JobListing, error) {
	// First, perform a GET request on the main landing page to harvest session cookies/CSRF token.
	req, err := http.NewRequest("GET", target.BaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token harvest request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token harvest request failed: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token harvest request returned bad status code: %d", resp.StatusCode)
	}

	u, err := url.Parse(target.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	targetURL := fmt.Sprintf("%s://%s/wday/cxs/%s/%s/jobs", u.Scheme, u.Host, target.Tenant, target.Site)
	return c.fetchJobsAt(targetURL, target.Category, target.Country, target.Location)
}

// resolveFacetID attempts to find the internal Workday ID for a given descriptor in a specific facet parameter.
func (c *WorkdayScraper) resolveFacetID(targetURL, facetParam, descriptor string) (string, error) {
	if descriptor == "" || descriptor == "All" {
		return "", nil
	}

	// Fetch all facets with 1 result limit to minimize data transfer
	reqPayload := WorkdayRequest{
		AppliedFacets: make(map[string][]string),
		Limit:         1,
		Offset:        0,
		SearchText:    "",
	}
	jsonData, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("facet resolution failed with status: %d", resp.StatusCode)
	}

	var workdayResp struct {
		Facets []struct {
			FacetParameter string `json:"facetParameter"`
			Values         []struct {
				Descriptor string `json:"descriptor"`
				ID         string `json:"id"`
				Values     []struct {
					Descriptor string `json:"descriptor"`
					ID         string `json:"id"`
					Values     []struct {
						Descriptor string `json:"descriptor"`
						ID         string `json:"id"`
					} `json:"values"`
				} `json:"values"`
			} `json:"values"`
		} `json:"facets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&workdayResp); err != nil {
		return "", err
	}

	// Search for the descriptor in the specified facet parameter (supporting 3 levels of nesting)
	for _, facet := range workdayResp.Facets {
		// Some tenants nest locations under locationMainGroup
		isLocationFacet := facet.FacetParameter == "locations" || facet.FacetParameter == "locationHierarchy1" || facet.FacetParameter == "locationCountry" || facet.FacetParameter == "locationMainGroup"
		isCategoryFacet := facet.FacetParameter == "jobFamilyGroup" || facet.FacetParameter == "functionalCategory"

		if (facetParam == "jobFamilyGroup" && isCategoryFacet) || (facetParam == "locations" && isLocationFacet) {
			for _, v := range facet.Values {
				if v.Descriptor == descriptor {
					return v.ID, nil
				}
				for _, v2 := range v.Values {
					if v2.Descriptor == descriptor {
						return v2.ID, nil
					}
					for _, v3 := range v2.Values {
						if v3.Descriptor == descriptor {
							return v3.ID, nil
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("descriptor '%s' not found in facet '%s'", descriptor, facetParam)
}

func (c *WorkdayScraper) fetchJobsAt(targetURL string, category, country, location string) ([]JobListing, error) {
	appliedFacets := make(map[string][]string)

	// Resolve Category ID
	if category != "" && category != "All" {
		id, err := c.resolveFacetID(targetURL, "jobFamilyGroup", category)
		if err == nil && id != "" {
			appliedFacets["jobFamilyGroup"] = []string{id}
		} else {
			// Fallback to descriptor if ID resolution fails (some tenants might accept it)
			appliedFacets["jobFamilyGroup"] = []string{category}
		}
	}

	// Resolve Country ID
	if country != "" && country != "All" {
		id, err := c.resolveFacetID(targetURL, "locations", country)
		if err == nil && id != "" {
			appliedFacets["locations"] = append(appliedFacets["locations"], id)
		} else {
			appliedFacets["locations"] = append(appliedFacets["locations"], country)
		}
	}

	// Resolve Location ID
	if location != "" && location != "All" {
		id, err := c.resolveFacetID(targetURL, "locations", location)
		if err == nil && id != "" {
			// Avoid duplicate IDs if country and location resolve to the same thing
			exists := false
			for _, existingID := range appliedFacets["locations"] {
				if existingID == id {
					exists = true
					break
				}
			}
			if !exists {
				appliedFacets["locations"] = append(appliedFacets["locations"], id)
			}
		} else {
			appliedFacets["locations"] = append(appliedFacets["locations"], location)
		}
	}

	reqPayload := WorkdayRequest{
		AppliedFacets: appliedFacets,
		Limit:         20,
		Offset:        0,
		SearchText:    "",
	}

	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", c.userAgent)

	// Add CSRF token from cookies if available
	if c.httpClient.Jar != nil {
		u, _ := url.Parse(targetURL)
		for _, cookie := range c.httpClient.Jar.Cookies(u) {
			if cookie.Name == "CALYPSO_CSRF_TOKEN" {
				req.Header.Set("X-Calypso-Csrf-Token", cookie.Value)
				break
			}
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Workday Error Body: %s\n", string(body))
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	var workdayResp WorkdayResponse
	if err := json.NewDecoder(resp.Body).Decode(&workdayResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return workdayResp.JobPostings, nil
}

package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
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
	// Log the discovery process
	log.Printf("Starting discovery for %s at %s", target.Name, targetURL)
	return c.fetchJobsAt(targetURL, target.Name, target.Category, target.Country, target.Location)
}

func mapFacetParameter(paramName string, firstLevelDescriptor string) string {
	if paramName == "locationMainGroup" {
		switch strings.ToLower(firstLevelDescriptor) {
		case "country":
			return "locationCountry"
		case "locations":
			return "locations"
		case "region":
			return "locationHierarchy1"
		}
	}
	return paramName
}

// resolveFacetID attempts to find the internal Workday ID and the correct facet parameter for a given descriptor.
func (c *WorkdayScraper) resolveFacetID(targetURL, facetType, descriptor string) (string, string, error) {
	if descriptor == "" || descriptor == "All" {
		return "", "", nil
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
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("facet resolution failed with status: %d", resp.StatusCode)
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
		return "", "", err
	}

	// Define which facet parameters are relevant for each facet type
	relevantParams := []string{}
	if facetType == "category" {
		relevantParams = []string{"jobFamilyGroup", "functionalCategory"}
	} else if facetType == "location" {
		relevantParams = []string{"locations", "locationHierarchy1", "locationCountry", "locationMainGroup"}
	}

	// First pass: Exact match in relevant facets
	for _, facet := range workdayResp.Facets {
		isRelevant := false
		for _, p := range relevantParams {
			if facet.FacetParameter == p {
				isRelevant = true
				break
			}
		}
		if !isRelevant {
			continue
		}

		// Search for the descriptor in the specified facet parameter (supporting 3 levels of nesting)
		for _, v := range facet.Values {
			if v.Descriptor == descriptor {
				return v.ID, mapFacetParameter(facet.FacetParameter, v.Descriptor), nil
			}
			for _, v2 := range v.Values {
				if v2.Descriptor == descriptor {
					return v2.ID, mapFacetParameter(facet.FacetParameter, v.Descriptor), nil
				}
				for _, v3 := range v2.Values {
					if v3.Descriptor == descriptor {
						return v3.ID, mapFacetParameter(facet.FacetParameter, v.Descriptor), nil
					}
				}
			}
		}
	}

	// Second pass: Partial match in relevant facets (case-insensitive)
	for _, facet := range workdayResp.Facets {
		isRelevant := false
		for _, p := range relevantParams {
			if facet.FacetParameter == p {
				isRelevant = true
				break
			}
		}
		if !isRelevant {
			continue
		}

		for _, v := range facet.Values {
			if strings.Contains(strings.ToLower(v.Descriptor), strings.ToLower(descriptor)) || strings.Contains(strings.ToLower(descriptor), strings.ToLower(v.Descriptor)) {
				return v.ID, mapFacetParameter(facet.FacetParameter, v.Descriptor), nil
			}
			for _, v2 := range v.Values {
				if strings.Contains(strings.ToLower(v2.Descriptor), strings.ToLower(descriptor)) || strings.Contains(strings.ToLower(descriptor), strings.ToLower(v2.Descriptor)) {
					return v2.ID, mapFacetParameter(facet.FacetParameter, v.Descriptor), nil
				}
				for _, v3 := range v2.Values {
					if strings.Contains(strings.ToLower(v3.Descriptor), strings.ToLower(descriptor)) || strings.Contains(strings.ToLower(descriptor), strings.ToLower(v3.Descriptor)) {
						return v3.ID, mapFacetParameter(facet.FacetParameter, v.Descriptor), nil
					}
				}
			}
		}
	}

	// Third pass: Exact match in ANY facet
	for _, facet := range workdayResp.Facets {
		for _, v := range facet.Values {
			if v.Descriptor == descriptor {
				return v.ID, mapFacetParameter(facet.FacetParameter, v.Descriptor), nil
			}
			for _, v2 := range v.Values {
				if v2.Descriptor == descriptor {
					return v2.ID, mapFacetParameter(facet.FacetParameter, v.Descriptor), nil
				}
				for _, v3 := range v2.Values {
					if v3.Descriptor == descriptor {
						return v3.ID, mapFacetParameter(facet.FacetParameter, v.Descriptor), nil
					}
				}
			}
		}
	}

	// Fourth pass: Partial match in ANY facet
	for _, facet := range workdayResp.Facets {
		for _, v := range facet.Values {
			if strings.Contains(strings.ToLower(v.Descriptor), strings.ToLower(descriptor)) || strings.Contains(strings.ToLower(descriptor), strings.ToLower(v.Descriptor)) {
				return v.ID, mapFacetParameter(facet.FacetParameter, v.Descriptor), nil
			}
			for _, v2 := range v.Values {
				if strings.Contains(strings.ToLower(v2.Descriptor), strings.ToLower(descriptor)) || strings.Contains(strings.ToLower(descriptor), strings.ToLower(v2.Descriptor)) {
					return v2.ID, mapFacetParameter(facet.FacetParameter, v.Descriptor), nil
				}
				for _, v3 := range v2.Values {
					if strings.Contains(strings.ToLower(v3.Descriptor), strings.ToLower(descriptor)) || strings.Contains(strings.ToLower(descriptor), strings.ToLower(v3.Descriptor)) {
						return v3.ID, mapFacetParameter(facet.FacetParameter, v.Descriptor), nil
					}
				}
			}
		}
	}

	return "", "", fmt.Errorf("descriptor '%s' not found in any facet", descriptor)
}

func (c *WorkdayScraper) fetchJobsAt(targetURL string, targetName, category, country, location string) ([]JobListing, error) {
	appliedFacets := make(map[string][]string)

	// Resolve Category ID
	if category != "" && category != "All" {
		id, param, err := c.resolveFacetID(targetURL, "category", category)
		if err == nil && id != "" {
			appliedFacets[param] = []string{id}
		}
	}

	// Resolve Country ID
	if country != "" && country != "All" {
		id, param, err := c.resolveFacetID(targetURL, "location", country)
		if err == nil && id != "" {
			appliedFacets[param] = append(appliedFacets[param], id)
		}
	}

	// Resolve Location ID
	if location != "" && location != "All" {
		id, param, err := c.resolveFacetID(targetURL, "location", location)
		if err == nil && id != "" {
			// Avoid duplicate IDs
			exists := false
			for _, existingID := range appliedFacets[param] {
				if existingID == id {
					exists = true
					break
				}
			}
			if !exists {
				appliedFacets[param] = append(appliedFacets[param], id)
			}
		}
	}

	// Pagination loop: fetch all pages of results
	const pageSize = 20
	var allListings []JobListing
	offset := 0
	total := -1 // Unknown until first response

	for {
		reqPayload := WorkdayRequest{
			AppliedFacets: appliedFacets,
			Limit:         pageSize,
			Offset:        offset,
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

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			log.Printf("Workday Error Status: %d", resp.StatusCode)
			log.Printf("Workday Error Headers: %v", resp.Header)
			log.Printf("Workday Error Body: %s", string(body))
			return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
		}

		var workdayResp WorkdayResponse
		if err := json.NewDecoder(resp.Body).Decode(&workdayResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		// Capture total from the first response
		if total < 0 {
			total = workdayResp.Total
		}

		allListings = append(allListings, workdayResp.JobPostings...)

		// Fetch descriptions for each job in the current page
		for i := len(allListings) - len(workdayResp.JobPostings); i < len(allListings); i++ {
			desc, err := c.fetchJobDescription(targetURL, allListings[i].ExternalPath)
			if err != nil {
				log.Printf("[%s] Warning: Failed to fetch description for job %s: %v", targetName, allListings[i].JobID, err)
				continue
			}
			allListings[i].Description = desc
		}

		log.Printf("[%s] Fetched page offset %d/%d...", targetName, offset, total)

		offset += pageSize
		if offset >= total {
			break
		}

		// Polite jittered backoff between page requests (200-500ms)
		jitter := time.Duration(200+rand.Intn(300)) * time.Millisecond
		time.Sleep(jitter)
	}

	return allListings, nil
}

func (c *WorkdayScraper) fetchJobDescription(jobsEndpoint, externalPath string) (string, error) {
	if externalPath == "" {
		return "", nil
	}
	// Workday jobs endpoint usually ends in /jobs
	// Detailed job endpoint is usually /job/externalPath
	detailURL := strings.Replace(jobsEndpoint, "/jobs", "/job"+externalPath, 1)

	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("detail request failed with status: %d", resp.StatusCode)
	}

	var fullJob WorkdayFullJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&fullJob); err != nil {
		return "", err
	}

	return fullJob.JobPostingInfo.JobDescription, nil
}

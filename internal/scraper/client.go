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
	maxRetries int
	retryDelay func(attempt int) time.Duration
	sleep      func(time.Duration)
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
		maxRetries: 2,
		retryDelay: func(attempt int) time.Duration {
			return time.Duration(attempt)*500*time.Millisecond +
				time.Duration(rand.Intn(250))*time.Millisecond
		},
		sleep: time.Sleep,
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
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

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

func mapFacetParameter(paramName, firstLevelDescriptor, facetType string) string {
	if paramName == "locationMainGroup" {
		switch strings.ToLower(firstLevelDescriptor) {
		case "country":
			return "locationCountry"
		case "locations":
			return "locations"
		case "region":
			return "locationHierarchy1"
		}

		// Some Workday tenants expose locationMainGroup as a flat list instead
		// of nesting values beneath Country, Locations, or Region headings.
		// In that shape, the caller's requested facet type is the only reliable
		// way to select the supported search parameter.
		switch facetType {
		case "country":
			return "locationCountry"
		case "location":
			return "locations"
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
	u, _ := url.Parse(targetURL)
	jsonData, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Origin", fmt.Sprintf("%s://%s", u.Scheme, u.Host))
	req.Header.Set("Referer", targetURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	// Add CSRF token from cookies if available
	if c.httpClient.Jar != nil {
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
				FacetParameter string `json:"facetParameter"`
				Descriptor     string `json:"descriptor"`
				ID             string `json:"id"`
				Values         []struct {
					FacetParameter string `json:"facetParameter"`
					Descriptor     string `json:"descriptor"`
					ID             string `json:"id"`
					Values         []struct {
						FacetParameter string `json:"facetParameter"`
						Descriptor     string `json:"descriptor"`
						ID             string `json:"id"`
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
	} else if facetType == "country" || facetType == "location" {
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
			valueParam := nestedFacetParameter(facet.FacetParameter, v.FacetParameter)
			if v.Descriptor == descriptor {
				return v.ID, mapFacetParameter(valueParam, v.Descriptor, facetType), nil
			}
			for _, v2 := range v.Values {
				valueParam2 := nestedFacetParameter(valueParam, v2.FacetParameter)
				if v2.Descriptor == descriptor {
					return v2.ID, mapFacetParameter(valueParam2, v.Descriptor, facetType), nil
				}
				for _, v3 := range v2.Values {
					valueParam3 := nestedFacetParameter(valueParam2, v3.FacetParameter)
					if v3.Descriptor == descriptor {
						return v3.ID, mapFacetParameter(valueParam3, v.Descriptor, facetType), nil
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
			valueParam := nestedFacetParameter(facet.FacetParameter, v.FacetParameter)
			if strings.Contains(strings.ToLower(v.Descriptor), strings.ToLower(descriptor)) || strings.Contains(strings.ToLower(descriptor), strings.ToLower(v.Descriptor)) {
				return v.ID, mapFacetParameter(valueParam, v.Descriptor, facetType), nil
			}
			for _, v2 := range v.Values {
				valueParam2 := nestedFacetParameter(valueParam, v2.FacetParameter)
				if strings.Contains(strings.ToLower(v2.Descriptor), strings.ToLower(descriptor)) || strings.Contains(strings.ToLower(descriptor), strings.ToLower(v2.Descriptor)) {
					return v2.ID, mapFacetParameter(valueParam2, v.Descriptor, facetType), nil
				}
				for _, v3 := range v2.Values {
					valueParam3 := nestedFacetParameter(valueParam2, v3.FacetParameter)
					if strings.Contains(strings.ToLower(v3.Descriptor), strings.ToLower(descriptor)) || strings.Contains(strings.ToLower(descriptor), strings.ToLower(v3.Descriptor)) {
						return v3.ID, mapFacetParameter(valueParam3, v.Descriptor, facetType), nil
					}
				}
			}
		}
	}

	// Third pass: Exact match in ANY facet
	for _, facet := range workdayResp.Facets {
		for _, v := range facet.Values {
			valueParam := nestedFacetParameter(facet.FacetParameter, v.FacetParameter)
			if v.Descriptor == descriptor {
				return v.ID, mapFacetParameter(valueParam, v.Descriptor, facetType), nil
			}
			for _, v2 := range v.Values {
				valueParam2 := nestedFacetParameter(valueParam, v2.FacetParameter)
				if v2.Descriptor == descriptor {
					return v2.ID, mapFacetParameter(valueParam2, v.Descriptor, facetType), nil
				}
				for _, v3 := range v2.Values {
					valueParam3 := nestedFacetParameter(valueParam2, v3.FacetParameter)
					if v3.Descriptor == descriptor {
						return v3.ID, mapFacetParameter(valueParam3, v.Descriptor, facetType), nil
					}
				}
			}
		}
	}

	// Fourth pass: Partial match in ANY facet
	for _, facet := range workdayResp.Facets {
		for _, v := range facet.Values {
			valueParam := nestedFacetParameter(facet.FacetParameter, v.FacetParameter)
			if strings.Contains(strings.ToLower(v.Descriptor), strings.ToLower(descriptor)) || strings.Contains(strings.ToLower(descriptor), strings.ToLower(v.Descriptor)) {
				return v.ID, mapFacetParameter(valueParam, v.Descriptor, facetType), nil
			}
			for _, v2 := range v.Values {
				valueParam2 := nestedFacetParameter(valueParam, v2.FacetParameter)
				if strings.Contains(strings.ToLower(v2.Descriptor), strings.ToLower(descriptor)) || strings.Contains(strings.ToLower(descriptor), strings.ToLower(v2.Descriptor)) {
					return v2.ID, mapFacetParameter(valueParam2, v.Descriptor, facetType), nil
				}
				for _, v3 := range v2.Values {
					valueParam3 := nestedFacetParameter(valueParam2, v3.FacetParameter)
					if strings.Contains(strings.ToLower(v3.Descriptor), strings.ToLower(descriptor)) || strings.Contains(strings.ToLower(descriptor), strings.ToLower(v3.Descriptor)) {
						return v3.ID, mapFacetParameter(valueParam3, v.Descriptor, facetType), nil
					}
				}
			}
		}
	}

	return "", "", fmt.Errorf("descriptor '%s' not found in any facet", descriptor)
}

func nestedFacetParameter(parentParam, childParam string) string {
	if childParam != "" {
		return childParam
	}
	return parentParam
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
		id, param, err := c.resolveFacetID(targetURL, "country", country)
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

		resp, err := c.executeSearchRequest(targetURL, jsonData, targetName, offset)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			log.Printf("Workday Error Status: %d", resp.StatusCode)
			log.Printf("Workday Error Headers: %v", resp.Header)
			log.Printf("Workday Error Body: %s", string(body))
			log.Printf("Request Payload: %s", string(jsonData))
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
			if allListings[i].ExternalPath != "" {
				applyURL, err := buildWorkdayDetailURL(targetURL, allListings[i].ExternalPath)
				if err == nil {
					allListings[i].ExternalPath = applyURL
				}
			}

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

func (c *WorkdayScraper) executeSearchRequest(targetURL string, jsonData []byte, targetName string, offset int) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequest("POST", targetURL, bytes.NewReader(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		c.applyAPIHeaders(req, targetURL)

		resp, err := c.httpClient.Do(req)
		if err == nil && !isRetryableWorkdayStatus(resp.StatusCode) {
			return resp, nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("bad status code: %d", resp.StatusCode)
		}

		if attempt == c.maxRetries {
			if err != nil {
				return nil, fmt.Errorf("request failed after %d attempts: %w", attempt+1, err)
			}
			return resp, nil
		}

		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		delay := c.retryDelay(attempt + 1)
		log.Printf(
			"[%s] Workday request offset %d failed (%v); retrying in %s (%d/%d)...",
			targetName,
			offset,
			lastErr,
			delay,
			attempt+1,
			c.maxRetries,
		)
		c.sleep(delay)
	}

	return nil, fmt.Errorf("request failed: %w", lastErr)
}

func (c *WorkdayScraper) applyAPIHeaders(req *http.Request, targetURL string) {
	u, _ := url.Parse(targetURL)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Origin", fmt.Sprintf("%s://%s", u.Scheme, u.Host))
	req.Header.Set("Referer", targetURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	if c.httpClient.Jar != nil {
		for _, cookie := range c.httpClient.Jar.Cookies(u) {
			if cookie.Name == "CALYPSO_CSRF_TOKEN" {
				req.Header.Set("X-Calypso-Csrf-Token", cookie.Value)
				break
			}
		}
	}
}

func isRetryableWorkdayStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

func (c *WorkdayScraper) fetchJobDescription(jobsEndpoint, externalPath string) (string, error) {
	if externalPath == "" {
		return "", nil
	}

	detailURL, err := buildWorkdayDetailURL(jobsEndpoint, externalPath)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("GET", detailURL, nil)
	if err != nil {
		return "", err
	}

	u, _ := url.Parse(jobsEndpoint)
	req.Header.Set("Accept", "application/json,application/xml,text/plain,*/*")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Referer", jobsEndpoint)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	// Add CSRF token from cookies if available
	if c.httpClient.Jar != nil {
		for _, cookie := range c.httpClient.Jar.Cookies(u) {
			if cookie.Name == "CALYPSO_CSRF_TOKEN" {
				req.Header.Set("X-Calypso-Csrf-Token", cookie.Value)
				break
			}
		}
	}

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

func buildWorkdayDetailURL(jobsEndpoint, externalPath string) (string, error) {
	if absolute, err := url.Parse(externalPath); err == nil && absolute.IsAbs() {
		return absolute.String(), nil
	}

	endpoint, err := url.Parse(jobsEndpoint)
	if err != nil {
		return "", fmt.Errorf("invalid Workday jobs endpoint: %w", err)
	}

	basePath := strings.TrimSuffix(endpoint.Path, "/jobs")
	detailPath := externalPath
	if !strings.HasPrefix(detailPath, "/") {
		detailPath = "/" + detailPath
	}
	if !strings.HasPrefix(detailPath, "/job/") {
		detailPath = "/job" + detailPath
	}

	endpoint.Path = strings.TrimRight(basePath, "/") + detailPath
	endpoint.RawPath = ""
	endpoint.RawQuery = ""
	endpoint.Fragment = ""
	return endpoint.String(), nil
}

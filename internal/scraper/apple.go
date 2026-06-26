package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type AppleScraper struct {
	Client  *http.Client
	BaseURL string
}

// NewAppleScraper creates a new AppleScraper instance.
func NewAppleScraper(client *http.Client) *AppleScraper {
	if client == nil {
		client = http.DefaultClient
	}
	return &AppleScraper{Client: client}
}

// AppleSearchResponse represents the structure of Apple's search API response.
type AppleSearchResponse struct {
	Res struct {
		SearchResults []AppleJob `json:"searchResults"`
		TotalRecords  int        `json:"totalRecords"`
	} `json:"res"`
}

type AppleJob struct {
	ID                      string          `json:"id"`
	JobPositionID           string          `json:"jobPositionId"`
	JobSummary              string          `json:"jobSummary"`
	PostingTitle            string          `json:"postingTitle"`
	PostingDate             string          `json:"postingDate"`
	TransformedPostingTitle string          `json:"transformedPostingTitle"`
	PositionID              string          `json:"positionId"`
	HomeOffice              bool            `json:"homeOffice"`
	Team                    AppleJobTeam    `json:"team"`
	Locations               []AppleLocation `json:"locations"`
}

type AppleJobTeam struct {
	TeamCode string `json:"teamCode"`
	TeamName string `json:"teamName"`
}

type AppleLocation struct {
	Name        string `json:"name"`
	City        string `json:"city"`
	State       string `json:"stateProvince"`
	CountryName string `json:"countryName"`
	CountryID   string `json:"countryID"`
}

// extractCookies manual parser function to bypass cookiejar issues.
func (a *AppleScraper) extractCookies(headers []string) map[string]string {
	cookies := make(map[string]string)
	cookieNames := []string{"jobs", "jssid", "cs-id", "AWSALBAPP-0", "AWSALBAPP-1", "AWSALBAPP-2", "AWSALBAPP-3"}
	for _, h := range headers {
		for _, name := range cookieNames {
			re := regexp.MustCompile(name + `=([^; ]+)`)
			if match := re.FindStringSubmatch(h); len(match) > 1 {
				cookies[name] = match[1]
			}
		}
	}
	return cookies
}

func (a *AppleScraper) formatCookieHeader(cookies map[string]string) string {
	var pairs []string
	for k, v := range cookies {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, "; ")
}

func (a *AppleScraper) getCountryLocationID(country string) string {
	countryMap := map[string]string{
		"us": "USA",
		"gb": "GBR",
		"uk": "GBR",
		"ca": "CAN",
		"in": "IND",
		"de": "DEU",
		"fr": "FRA",
		"jp": "JPN",
		"cn": "CHN",
		"au": "AUS",
		"br": "BRA",
	}

	c := strings.ToLower(strings.TrimSpace(country))
	if c == "all" || c == "" {
		return "postLocation-USA" // Default to USA to yield listings
	}

	if val, ok := countryMap[c]; ok {
		return "postLocation-" + val
	}

	if len(c) == 3 {
		return "postLocation-" + strings.ToUpper(c)
	}

	return "postLocation-USA"
}

func (a *AppleScraper) getCountryISOCode(country string) string {
	countryMap := map[string]string{
		"us":             "USA",
		"united states":  "USA",
		"gb":             "GBR",
		"uk":             "GBR",
		"united kingdom": "GBR",
		"ca":             "CAN",
		"canada":         "CAN",
		"in":             "IND",
		"india":          "IND",
		"de":             "DEU",
		"germany":        "DEU",
		"fr":             "FRA",
		"france":         "FRA",
		"jp":             "JPN",
		"japan":          "JPN",
		"cn":             "CHN",
		"china":          "CHN",
		"au":             "AUS",
		"australia":      "AUS",
		"br":             "BRA",
		"brazil":         "BRA",
	}

	c := strings.ToLower(strings.TrimSpace(country))
	if val, ok := countryMap[c]; ok {
		return val
	}

	if len(c) == 3 {
		return strings.ToUpper(c)
	}

	return "USA"
}


func (a *AppleScraper) FetchJobs(target TargetCompany) ([]JobListing, error) {
	targetCategory := stripNumericPrefix(target.Category)
	targetLocation := stripNumericPrefix(target.Location)
	targetCountry := stripNumericPrefix(target.Country)

	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	allCookies := make(map[string]string)

	baseURL := a.BaseURL
	if baseURL == "" {
		baseURL = "https://jobs.apple.com"
	}

	// 1. GET Landing Page
	landingURL := baseURL + "/en-us/search"
	reqGet, err := http.NewRequest("GET", landingURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create landing page request: %w", err)
	}
	reqGet.Header.Set("User-Agent", userAgent)
	reqGet.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	respGet, err := a.Client.Do(reqGet)
	if err != nil {
		return nil, fmt.Errorf("landing page request failed: %w", err)
	}
	for k, v := range a.extractCookies(respGet.Header["Set-Cookie"]) {
		allCookies[k] = v
	}
	respGet.Body.Close()

	// 2. GET CSRF Token
	csrfURL := baseURL + "/api/v1/CSRFToken"
	reqToken, err := http.NewRequest("GET", csrfURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSRFToken request: %w", err)
	}
	reqToken.Header.Set("User-Agent", userAgent)
	reqToken.Header.Set("Referer", landingURL)
	cookieHeaderVal := a.formatCookieHeader(allCookies)
	if cookieHeaderVal != "" {
		reqToken.Header.Set("Cookie", cookieHeaderVal)
	}

	respToken, err := a.Client.Do(reqToken)
	if err != nil {
		return nil, fmt.Errorf("CSRFToken request failed: %w", err)
	}
	csrfToken := respToken.Header.Get("X-Apple-CSRF-Token")
	for k, v := range a.extractCookies(respToken.Header["Set-Cookie"]) {
		allCookies[k] = v
	}
	respToken.Body.Close()

	if csrfToken == "" {
		return nil, fmt.Errorf("failed to harvest CSRF token from jobs.apple.com")
	}

	// Determine initial search parameters
	queryKeyword := ""
	if targetCategory != "" && !strings.EqualFold(targetCategory, "all") {
		queryKeyword = targetCategory
	}

	locationFilter := a.getCountryLocationID(targetCountry)

	var listings []JobListing
	page := 1

	for {
		payload := map[string]interface{}{
			"query":    queryKeyword,
			"locale":   "en-us",
			"page":     page,
			"sort":     "newest",
			"filters": map[string]interface{}{
				"locations": []string{locationFilter},
			},
			"format": map[string]interface{}{
				"longDate":   "MMMM D, YYYY",
				"mediumDate": "MMM D, YYYY",
			},
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal search request payload: %w", err)
		}

		reqPost, err := http.NewRequest("POST", baseURL+"/api/v1/search", bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create search post request: %w", err)
		}
		reqPost.Header.Set("Content-Type", "application/json")
		reqPost.Header.Set("Accept", "application/json, text/plain, */*")
		reqPost.Header.Set("User-Agent", userAgent)
		reqPost.Header.Set("Origin", baseURL)
		reqPost.Header.Set("Referer", landingURL)
		reqPost.Header.Set("X-Requested-With", "XMLHttpRequest")
		reqPost.Header.Set("Accept-Language", "en-US,en;q=0.9")
		reqPost.Header.Set("Cookie", a.formatCookieHeader(allCookies))
		reqPost.Header["X-Apple-CSRF-Token"] = []string{csrfToken}
		reqPost.Header["browserLocale"] = []string{"en-us"}

		respPost, err := a.Client.Do(reqPost)
		if err != nil {
			return nil, fmt.Errorf("search POST request failed at page %d: %w", page, err)
		}

		bodyBytes, err := io.ReadAll(respPost.Body)
		respPost.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read search response body at page %d: %w", page, err)
		}

		if respPost.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("apple search api returned status %d at page %d: %s", respPost.StatusCode, page, string(bodyBytes))
		}

		var searchResp AppleSearchResponse
		if err := json.Unmarshal(bodyBytes, &searchResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal search response at page %d: %w", page, err)
		}

		totalRecords := searchResp.Res.TotalRecords
		jobs := searchResp.Res.SearchResults

		if len(jobs) == 0 {
			break
		}

		for _, job := range jobs {
			// Apply category/keyword checks
			categoryMatched := true
			if queryKeyword != "" {
				titleLower := strings.ToLower(job.PostingTitle)
				teamLower := strings.ToLower(job.Team.TeamName)
				catLower := strings.ToLower(queryKeyword)
				if !strings.Contains(titleLower, catLower) && !strings.Contains(teamLower, catLower) {
					categoryMatched = false
				}
			}

			// Apply location checks
			locMatched := true
			if targetLocation != "" && !strings.EqualFold(targetLocation, "all") {
				targetLocLower := strings.ToLower(strings.TrimSpace(targetLocation))
				var locNames []string
				for _, loc := range job.Locations {
					locNames = append(locNames, strings.ToLower(loc.Name), strings.ToLower(loc.City))
				}
				locsStr := strings.Join(locNames, " ")

				if strings.Contains(targetLocLower, "remote") {
					if !strings.Contains(locsStr, "remote") && !job.HomeOffice {
						locMatched = false
					}
				} else {
					if !strings.Contains(locsStr, targetLocLower) {
						locMatched = false
					}
				}
			} else if targetCountry != "" && !strings.EqualFold(targetCountry, "all") {
				targetCountryISO := a.getCountryISOCode(targetCountry)
				countryMatchedForJob := false
				for _, loc := range job.Locations {
					if strings.Contains(strings.ToUpper(loc.CountryID), targetCountryISO) ||
						strings.Contains(strings.ToLower(loc.CountryName), strings.ToLower(targetCountry)) {
						countryMatchedForJob = true
						break
					}
				}
				if !countryMatchedForJob {
					locMatched = false
				}
			}

			if categoryMatched && locMatched {
				var locTexts []string
				for _, loc := range job.Locations {
					locPart := loc.Name
					if loc.State != "" {
						locPart += ", " + loc.State
					}
					if loc.CountryName != "" && loc.CountryName != loc.Name {
						locPart += " (" + loc.CountryName + ")"
					}
					locTexts = append(locTexts, locPart)
				}
				locationText := strings.Join(locTexts, "; ")

				transformedTitle := job.TransformedPostingTitle
				if transformedTitle == "" {
					transformedTitle = strings.ToLower(strings.ReplaceAll(job.PostingTitle, " ", "-"))
				}
				externalPath := fmt.Sprintf("https://jobs.apple.com/en-us/details/%s/%s", job.PositionID, transformedTitle)

				listings = append(listings, JobListing{
					JobID:         fmt.Sprintf("apple-%s", job.PositionID),
					Title:         job.PostingTitle,
					LocationsText: locationText,
					PostedOn:      job.PostingDate,
					ExternalPath:  externalPath,
					Description:   job.JobSummary,
				})
			}
		}

		if len(listings) >= totalRecords || (page*20) >= totalRecords {
			break
		}

		page++

		// Jittered backoff delay of 200–500ms
		delayMs := 200 + rand.Intn(300)
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	if Debug {
		log.Printf("Successfully scraped %d Apple jobs matching criteria", len(listings))
	}

	return listings, nil
}

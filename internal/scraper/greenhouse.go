package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type GreenhouseScraper struct {
	Client  *http.Client
	BaseURL string
}

// GreenhouseResponse maps the public Greenhouse payload structure
type GreenhouseResponse struct {
	Jobs []struct {
		ID       int64  `json:"id"`
		Title    string `json:"title"`
		Content  string `json:"content"` // Contains the full job description HTML
		Location struct {
			Name string `json:"name"`
		} `json:"location"`
		AbsoluteURL string `json:"absolute_url"`
		Departments []struct {
			Name string `json:"name"`
		} `json:"departments"`
	} `json:"jobs"`
}

var prefixRegex = regexp.MustCompile(`^\d+\.\s*`)

func stripNumericPrefix(s string) string {
	return prefixRegex.ReplaceAllString(s, "")
}

func (g *GreenhouseScraper) FetchJobs(target TargetCompany) ([]JobListing, error) {
	// Clean selection prefixes if TUI passes them
	targetCategory := stripNumericPrefix(target.Category)
	targetLocation := stripNumericPrefix(target.Location)
	targetCountry := stripNumericPrefix(target.Country)

	baseURL := g.BaseURL
	if baseURL == "" {
		baseURL = "https://boards-api.greenhouse.io"
	}
	// Fetch all jobs (with content=true to retrieve departments details)
	endpoint := fmt.Sprintf("%s/v1/boards/%s/jobs?content=true", baseURL, target.Tenant)

	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("User-Agent", "openHunt/2.0")

	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("greenhouse api returned status: %d", resp.StatusCode)
	}

	var ghResp GreenhouseResponse
	if err := json.NewDecoder(resp.Body).Decode(&ghResp); err != nil {
		return nil, err
	}

	// Normalize data into our universal struct format with loose filtering
	var listings []JobListing
	for _, ghJob := range ghResp.Jobs {
		// Category Check (Department)
		categoryMatched := true
		var depNames []string
		for _, d := range ghJob.Departments {
			depNames = append(depNames, d.Name)
		}
		jobDepartment := strings.Join(depNames, ", ")

		categoryMatched = MatchCategory(jobDepartment, targetCategory)
		if !categoryMatched && Debug {
			log.Printf("Skipping %s due to category mismatch: %s (expected %s)", ghJob.Title, jobDepartment, targetCategory)
		}

		locationMatched := MatchLocation(ghJob.Location.Name, targetLocation, targetCountry)
		if !locationMatched && Debug {
			log.Printf("Skipping %s due to location mismatch: %s (expected %s / country %s)", ghJob.Title, ghJob.Location.Name, targetLocation, targetCountry)
		}

		if categoryMatched && locationMatched {
			listings = append(listings, JobListing{
				JobID:         fmt.Sprintf("gh-%d", ghJob.ID), // Prefix to avoid global database constraints
				Title:         ghJob.Title,
				LocationsText: ghJob.Location.Name,
				ExternalPath:  ghJob.AbsoluteURL,
				Description:   ghJob.Content,
			})
		}
	}

	return listings, nil
}

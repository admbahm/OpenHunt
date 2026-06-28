package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type LeverScraper struct {
	Client  *http.Client
	BaseURL string
}

type LeverResponse []struct {
	ID         string `json:"id"`
	Title      string `json:"title"` // Fallback if they ever send "title"
	Text       string `json:"text"`  // This is the actual job title from Lever
	Categories struct {
		Department string   `json:"department"`
		Team       string   `json:"team"`
		Location   string   `json:"location"`
		AllLocations []string `json:"allLocations"`
	} `json:"categories"`
	WorkplaceType string `json:"workplaceType"`
	HostedURL     string `json:"hostedUrl"`
	Description   string `json:"description"`
}

func (s *LeverScraper) FetchJobs(target TargetCompany) ([]JobListing, error) {
	if s.Client == nil {
		s.Client = http.DefaultClient
	}

	targetCategory := stripNumericPrefix(target.Category)
	targetLocation := stripNumericPrefix(target.Location)
	targetCountry := stripNumericPrefix(target.Country)

	baseURL := s.BaseURL
	if baseURL == "" {
		baseURL = "https://api.lever.co"
	}
	endpoint := fmt.Sprintf("%s/v0/postings/%s?mode=json", baseURL, target.Tenant)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "openHunt/2.0")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lever api returned status: %d", resp.StatusCode)
	}

	var leverResp LeverResponse
	if err := json.NewDecoder(resp.Body).Decode(&leverResp); err != nil {
		return nil, err
	}

	var listings []JobListing
	for _, job := range leverResp {
		title := job.Title
		if title == "" {
			title = job.Text
		}

		// Department matching
		deptParts := []string{}
		if job.Categories.Department != "" {
			deptParts = append(deptParts, job.Categories.Department)
		}
		if job.Categories.Team != "" && job.Categories.Team != job.Categories.Department {
			deptParts = append(deptParts, job.Categories.Team)
		}
		jobDept := strings.Join(deptParts, ", ")

		categoryMatched := MatchCategory(jobDept, targetCategory)

		// Location matching
		locParts := []string{}
		if len(job.Categories.AllLocations) > 0 {
			locParts = append(locParts, job.Categories.AllLocations...)
		} else if job.Categories.Location != "" {
			locParts = append(locParts, job.Categories.Location)
		}
		if job.WorkplaceType != "" {
			// Avoid duplicates
			found := false
			for _, l := range locParts {
				if strings.EqualFold(l, job.WorkplaceType) {
					found = true
					break
				}
			}
			if !found {
				locParts = append(locParts, job.WorkplaceType)
			}
		}
		jobLoc := strings.Join(locParts, ", ")

		locationMatched := MatchLocation(jobLoc, targetLocation, targetCountry)

		if !categoryMatched || !locationMatched {
			continue
		}

		listings = append(listings, JobListing{
			JobID:         fmt.Sprintf("lever-%s", job.ID),
			Title:         title,
			LocationsText: jobLoc,
			ExternalPath:  job.HostedURL,
			Description:   job.Description,
		})
	}

	return listings, nil
}

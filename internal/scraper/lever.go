package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type LeverScraper struct {
	Client  *http.Client
	BaseURL string
}

type LeverResponse []struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Categories struct {
		Team string `json:"team"`
	} `json:"categories"`
	WorkplaceType string `json:"workplaceType"`
	HostedURL     string `json:"hostedUrl"`
	Description   string `json:"description"`
}

func (s *LeverScraper) FetchJobs(target TargetCompany) ([]JobListing, error) {
	if s.Client == nil {
		s.Client = http.DefaultClient
	}

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
		listings = append(listings, JobListing{
			JobID:         fmt.Sprintf("lever-%s", job.ID),
			Title:         job.Title,
			LocationsText: job.WorkplaceType, // Lever often puts remote/hybrid here or in a separate location field
			ExternalPath:  job.HostedURL,
			Description:   job.Description,
		})
	}

	return listings, nil
}

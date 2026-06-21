package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type GreenhouseScraper struct {
	Client *http.Client
}

// GreenhouseResponse maps the public Greenhouse payload structure
type GreenhouseResponse struct {
	Jobs []struct {
		ID       int64  `json:"id"`
		Title    string `json:"title"`
		Location struct {
			Name string `json:"name"`
		} `json:"location"`
		AbsoluteURL string `json:"absolute_url"`
	} `json:"jobs"`
}

func (g *GreenhouseScraper) FetchJobs(target TargetCompany) ([]JobListing, error) {
	// Greenhouse uses a singular company token/tenant name
	endpoint := fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs", target.Tenant)

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

	// Normalize data into our universal struct format
	listings := make([]JobListing, len(ghResp.Jobs))
	for i, ghJob := range ghResp.Jobs {
		listings[i] = JobListing{
			JobID:         fmt.Sprintf("gh-%d", ghJob.ID), // Prefix to avoid global database constraints
			Title:         ghJob.Title,
			LocationsText: ghJob.Location.Name,
			ExternalPath:  ghJob.AbsoluteURL,
		}
	}

	return listings, nil
}

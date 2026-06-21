package scraper

import "encoding/json"

// TargetCompany represents a company whose job board we want to scrape.
type TargetCompany struct {
	Name     string `db:"name"`
	Tenant   string `db:"tenant"`
	Site     string `db:"site"`     // Note: This will be empty for Greenhouse/Lever
	BaseURL  string `db:"base_url"` // Main landing page
	Platform string `db:"platform"` // 'workday', 'greenhouse', 'lever', etc.
}

// JobListing standardizes data across all ATS vendors for the SQLite diff engine
type JobListing struct {
	JobID         string `json:"id"`
	Title         string `json:"title"`
	LocationsText string `json:"location"`
	PostedOn      string `json:"posted_on"` // Standardized naming might be better but keeping consistency for now
	ExternalPath  string `json:"url"`
}

// JobScraper defines the single behavioral contract for all ingestion backends
type JobScraper interface {
	FetchJobs(target TargetCompany) ([]JobListing, error)
}

// UnmarshalJSON customizes unmarshaling to extract a unique JobID.
func (j *JobListing) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Title         string `json:"title"`
		LocationsText string `json:"locationsText"`
		PostedOn      string `json:"postedOn"`
		ExternalPath  string `json:"externalPath"`
	}
	aux := &struct {
		BulletinNumber string   `json:"bulletinNumber"`
		BulletFields   []string `json:"bulletFields"`
		*Alias
	}{
		Alias: &Alias{},
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	j.Title = aux.Alias.Title
	j.LocationsText = aux.Alias.LocationsText
	j.PostedOn = aux.Alias.PostedOn
	j.ExternalPath = aux.Alias.ExternalPath

	if aux.BulletinNumber != "" {
		j.JobID = aux.BulletinNumber
	} else if len(aux.BulletFields) > 0 && aux.BulletFields[0] != "" {
		j.JobID = aux.BulletFields[0]
	} else {
		j.JobID = j.ExternalPath
	}
	return nil
}

// WorkdayResponse represents the top-level structure of the Workday CXS API response.
type WorkdayResponse struct {
	JobPostings []JobListing `json:"jobPostings"`
	Total       int          `json:"total"`
}

// WorkdayRequest represents the payload for the Workday CXS API POST request.
type WorkdayRequest struct {
	AppliedFacets map[string][]string `json:"appliedFacets"`
	Limit         int                 `json:"limit"`
	Offset        int                 `json:"offset"`
	SearchText    string              `json:"searchText"`
}

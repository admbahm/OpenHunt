package scraper

import "encoding/json"

// TargetCompany represents a company whose job board we want to scrape.
type TargetCompany struct {
	Name    string
	Tenant  string
	Site    string
	BaseURL string
}

// JobListing captures the essential fields from a Workday CXS job posting.
type JobListing struct {
	JobID         string `json:"-"`
	Title         string `json:"title"`
	LocationsText string `json:"locationsText"`
	PostedOn      string `json:"postedOn"`
	ExternalPath  string `json:"externalPath"`
}

// UnmarshalJSON customizes unmarshaling to extract a unique JobID.
func (j *JobListing) UnmarshalJSON(data []byte) error {
	type Alias JobListing
	aux := &struct {
		BulletinNumber string   `json:"bulletinNumber"`
		BulletFields   []string `json:"bulletFields"`
		*Alias
	}{
		Alias: (*Alias)(j),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.BulletinNumber != "" {
		j.JobID = aux.BulletinNumber
	} else if len(aux.BulletFields) > 0 && aux.BulletFields[0] != "" {
		j.JobID = aux.BulletFields[0]
	} else {
		j.JobID = aux.ExternalPath
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

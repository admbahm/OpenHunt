package scraper

// TargetCompany represents a company whose job board we want to scrape.
type TargetCompany struct {
	Name    string
	Tenant  string
	Site    string
	BaseURL string
}

// JobListing captures the essential fields from a Workday CXS job posting.
type JobListing struct {
	JobID         string `json:"bulletinNumber"`
	Title         string `json:"title"`
	LocationsText string `json:"locationsText"`
	PostedOn      string `json:"postedOn"`
	ExternalPath  string `json:"externalPath"`
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

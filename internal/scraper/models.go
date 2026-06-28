package scraper

import (
	"encoding/json"
	"strings"
)

// TargetCompany represents a company whose job board we want to scrape.
type TargetCompany struct {
	Name     string `db:"name"`
	Tenant   string `db:"tenant"`
	Site     string `db:"site"`     // Note: This will be empty for Greenhouse/Lever
	BaseURL  string `db:"base_url"` // Main landing page
	Platform string `db:"platform"` // 'workday', 'greenhouse', 'lever', etc.
	Category string `db:"-"`        // Runtime filter
	Country  string `db:"-"`        // Runtime filter
	Location string `db:"-"`        // Runtime filter
}

// JobListing standardizes data across all ATS vendors for the SQLite diff engine
type JobListing struct {
	JobID         string `json:"id"`
	Title         string `json:"title"`
	LocationsText string `json:"location"`
	PostedOn      string `json:"posted_on"` // Standardized naming might be better but keeping consistency for now
	ExternalPath  string `json:"url"`
	Description   string `json:"description"`
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
	j.Description = "" // Will be populated later or via individual fetch

	if aux.BulletinNumber != "" {
		j.JobID = aux.BulletinNumber
	} else if len(aux.BulletFields) > 0 && aux.BulletFields[0] != "" {
		j.JobID = aux.BulletFields[0]
	} else {
		j.JobID = j.ExternalPath
	}
	return nil
}

var usStates = map[string]bool{
	"al": true, "ak": true, "az": true, "ar": true, "ca": true, "co": true, "ct": true, "de": true, "fl": true, "ga": true,
	"hi": true, "id": true, "il": true, "in": true, "ia": true, "ks": true, "ky": true, "la": true, "me": true, "md": true,
	"ma": true, "mi": true, "mn": true, "ms": true, "mo": true, "mt": true, "ne": true, "nv": true, "nh": true, "nj": true,
	"nm": true, "ny": true, "nc": true, "nd": true, "oh": true, "ok": true, "or": true, "pa": true, "ri": true, "sc": true,
	"sd": true, "tn": true, "tx": true, "ut": true, "vt": true, "va": true, "wa": true, "wv": true, "wi": true, "wy": true,
}

// MatchLocation matches a job location against the user's location and country filters loosely.
func MatchLocation(jobLocation, targetLocation, targetCountry string) bool {
	jobLocLower := strings.ToLower(jobLocation)

	if targetLocation != "" && strings.ToLower(targetLocation) != "all" {
		targetLocLower := strings.ToLower(strings.TrimSpace(targetLocation))

		if strings.Contains(targetLocLower, "remote") {
			return strings.Contains(jobLocLower, "remote")
		}

		cityPart := targetLocLower
		if idx := strings.Index(targetLocLower, ","); idx != -1 {
			cityPart = strings.TrimSpace(targetLocLower[:idx])
		}

		return strings.Contains(jobLocLower, cityPart)
	}

	if targetCountry != "" && strings.ToLower(targetCountry) != "all" {
		targetCountryLower := strings.ToLower(targetCountry)
		if strings.Contains(targetCountryLower, "united states") || strings.Contains(targetCountryLower, "usa") {
			if strings.Contains(jobLocLower, "united states") ||
				strings.Contains(jobLocLower, "usa") ||
				strings.Contains(jobLocLower, "u.s.") ||
				strings.Contains(jobLocLower, "america") {
				return true
			}
			// Check for two-letter US state suffix (e.g., ", ca", ", tx")
			parts := strings.Split(jobLocLower, ",")
			if len(parts) > 1 {
				lastPart := strings.TrimSpace(parts[len(parts)-1])
				if len(lastPart) == 2 && usStates[lastPart] {
					return true
				}
			}
			// Check space-separated suffix
			words := strings.Fields(jobLocLower)
			if len(words) > 0 {
				lastWord := words[len(words)-1]
				if len(lastWord) == 2 && usStates[lastWord] {
					return true
				}
			}
			return false
		}
		return strings.Contains(jobLocLower, targetCountryLower)
	}

	return true
}

// MatchCategory matches a job category (department) against the user's category filter.
func MatchCategory(jobCategory, targetCategory string) bool {
	if targetCategory == "" || strings.ToLower(targetCategory) == "all" {
		return true
	}
	return strings.Contains(strings.ToLower(jobCategory), strings.ToLower(targetCategory))
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

// WorkdayFullJobResponse represents the response when fetching a single job's details.
type WorkdayFullJobResponse struct {
	JobPostingInfo struct {
		JobDescription string `json:"jobDescription"`
	} `json:"jobPostingInfo"`
}

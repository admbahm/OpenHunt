package scraper

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type AshbyScraper struct {
	Client  *http.Client
	BaseURL string
}

type AshbyResponse struct {
	JobBoard struct {
		JobPostings []AshbyPosting `json:"jobPostings"`
	} `json:"jobBoard"`
	Posting *AshbyPosting `json:"posting"`
}

type AshbyPosting struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	DepartmentName  string `json:"departmentName"`
	TeamName        string `json:"teamName"`
	LocationName    string `json:"locationName"`
	WorkplaceType   string `json:"workplaceType"`
	PublishedDate   string `json:"publishedDate"`
	DescriptionHTML string `json:"descriptionHtml"`
	ExternalLink    string `json:"externalLink"`
}

func extractAshbyAppData(r io.Reader) (*AshbyResponse, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	jsonData, err := extractAshbyAppDataJSON(string(body))
	if err != nil {
		return nil, fmt.Errorf("ashby page did not include app data")
	}

	var ashbyResp AshbyResponse
	if err := json.Unmarshal([]byte(jsonData), &ashbyResp); err != nil {
		return nil, err
	}
	return &ashbyResp, nil
}

func extractAshbyAppDataJSON(body string) (string, error) {
	const marker = "window.__appData"
	markerIndex := strings.Index(body, marker)
	if markerIndex == -1 {
		return "", fmt.Errorf("missing window.__appData")
	}

	assignment := body[markerIndex+len(marker):]
	equalsIndex := strings.Index(assignment, "=")
	if equalsIndex == -1 {
		return "", fmt.Errorf("missing assignment")
	}

	data := assignment[equalsIndex+1:]
	start := strings.Index(data, "{")
	if start == -1 {
		return "", fmt.Errorf("missing json object")
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(data); i++ {
		ch := data[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return data[start : i+1], nil
			}
		}
	}

	return "", fmt.Errorf("unterminated json object")
}

func ashbyPostingURL(baseURL, tenant, postingID string) string {
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(baseURL, "/"), tenant, postingID)
}

func ashbyJobLocation(job AshbyPosting) string {
	switch {
	case job.LocationName != "" && job.WorkplaceType != "":
		return fmt.Sprintf("%s (%s)", job.LocationName, job.WorkplaceType)
	case job.LocationName != "":
		return job.LocationName
	default:
		return job.WorkplaceType
	}
}

func ashbyJobDepartment(job AshbyPosting) string {
	parts := []string{}
	if job.DepartmentName != "" {
		parts = append(parts, job.DepartmentName)
	}
	if job.TeamName != "" && job.TeamName != job.DepartmentName {
		parts = append(parts, job.TeamName)
	}
	return strings.Join(parts, ", ")
}

func ashbyDescriptionText(descriptionHTML string) string {
	descriptionHTML = strings.ReplaceAll(descriptionHTML, "</p>", "\n")
	descriptionHTML = strings.ReplaceAll(descriptionHTML, "</li>", "\n")
	descriptionHTML = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(descriptionHTML, "")
	return strings.TrimSpace(html.UnescapeString(descriptionHTML))
}

func (s *AshbyScraper) fetchPostingDescription(baseURL, tenant, postingID string) string {
	req, err := http.NewRequest("GET", ashbyPostingURL(baseURL, tenant, postingID), nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "openHunt/2.0")

	resp, err := s.Client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	ashbyResp, err := extractAshbyAppData(resp.Body)
	if err != nil || ashbyResp.Posting == nil {
		return ""
	}
	return ashbyDescriptionText(ashbyResp.Posting.DescriptionHTML)
}

func (s *AshbyScraper) FetchJobs(target TargetCompany) ([]JobListing, error) {
	if s.Client == nil {
		s.Client = http.DefaultClient
	}

	targetCategory := stripNumericPrefix(target.Category)
	targetLocation := stripNumericPrefix(target.Location)
	targetCountry := stripNumericPrefix(target.Country)

	baseURL := s.BaseURL
	if baseURL == "" {
		baseURL = "https://jobs.ashbyhq.com"
	}

	endpoint := fmt.Sprintf("%s/%s", strings.TrimRight(baseURL, "/"), target.Tenant)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "openHunt/2.0")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ashby public board returned status: %d", resp.StatusCode)
	}

	ashbyResp, err := extractAshbyAppData(resp.Body)
	if err != nil {
		return nil, err
	}

	var listings []JobListing
	for _, job := range ashbyResp.JobBoard.JobPostings {
		department := ashbyJobDepartment(job)
		location := ashbyJobLocation(job)

		categoryMatched := MatchCategory(department, targetCategory)
		locationMatched := MatchLocation(location, targetLocation, targetCountry)

		if !categoryMatched || !locationMatched {
			continue
		}

		externalPath := job.ExternalLink
		if externalPath == "" {
			externalPath = ashbyPostingURL(baseURL, target.Tenant, job.ID)
		}

		description := ashbyDescriptionText(job.DescriptionHTML)
		if description == "" {
			description = s.fetchPostingDescription(baseURL, target.Tenant, job.ID)
		}

		listings = append(listings, JobListing{
			JobID:         fmt.Sprintf("ashby-%s", job.ID),
			Title:         job.Title,
			LocationsText: location,
			PostedOn:      job.PublishedDate,
			ExternalPath:  externalPath,
			Description:   description,
		})
	}

	return listings, nil
}

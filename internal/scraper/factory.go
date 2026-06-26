package scraper

import (
	"fmt"
	"net/http"
)

// NewScraperFactory instantiates the correct structural engine based on database state
func NewScraperFactory(platform string, httpClient *http.Client) (JobScraper, error) {
	switch platform {
	case "workday":
		return NewWorkdayScraper(httpClient), nil
	case "greenhouse":
		if httpClient == nil {
			httpClient = http.DefaultClient
		}
		return &GreenhouseScraper{Client: httpClient}, nil
	case "lever":
		return &LeverScraper{Client: httpClient}, nil
	case "ashby":
		return &AshbyScraper{Client: httpClient}, nil
	case "apple":
		if httpClient == nil {
			httpClient = http.DefaultClient
		}
		return &AppleScraper{Client: httpClient}, nil
	default:
		return nil, fmt.Errorf("unsupported scraping platform: %s", platform)
	}
}

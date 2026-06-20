package scraper

import (
	"log"
	"sync"
)

// Result carries the outcome of a single company scrape.
type Result struct {
	CompanyName string
	Jobs        []JobListing
	Error       error
}

// Scraper orchestrates concurrent job scraping.
type Scraper struct {
	client      *Client
	workerCount int
}

// NewScraper initializes a new Scraper.
func NewScraper(workerCount int) *Scraper {
	return &Scraper{
		client:      NewClient(),
		workerCount: workerCount,
	}
}

// Run executes the scraping process for a list of target companies.
func (s *Scraper) Run(companies []TargetCompany) []Result {
	jobsChan := make(chan TargetCompany, len(companies))
	resultsChan := make(chan Result, len(companies))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < s.workerCount; i++ {
		wg.Add(1)
		go s.worker(&wg, jobsChan, resultsChan)
	}

	// Feed companies into the jobs channel
	for _, company := range companies {
		jobsChan <- company
	}
	close(jobsChan)

	// Close results channel once all workers are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	var results []Result
	for res := range resultsChan {
		results = append(results, res)
	}

	return results
}

// worker processes companies from the jobs channel.
func (s *Scraper) worker(wg *sync.WaitGroup, jobs <-chan TargetCompany, results chan<- Result) {
	defer wg.Done()

	for company := range jobs {
		log.Printf("Scraping %s...", company.Name)
		jobsList, err := s.client.FetchJobs(company)

		results <- Result{
			CompanyName: company.Name,
			Jobs:        jobsList,
			Error:       err,
		}
	}
}

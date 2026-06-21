package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/openhunt/openhunt/internal/db"
	"github.com/openhunt/openhunt/internal/scraper"
	"github.com/openhunt/openhunt/internal/telemetry"
	"github.com/openhunt/openhunt/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

type PipelineJob struct {
	Company scraper.TargetCompany
	Job     scraper.JobListing
}

func main() {
	fmt.Println("Starting openHunt...")

	// Initialize the database
	dbPath := "database/openhunt.db"
	store, err := db.NewSQLStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer store.Close()

	// Seed target companies
	if err := store.SeedTargets(); err != nil {
		log.Fatalf("Failed to seed target companies: %v", err)
	}

	// TUI Selection
	cats := []string{"All", "Engineering", "Quality", "Information Technology", "Sales", "Manufacturing and Operations"}
	countries := []string{"All", "United States of America", "Ireland", "India", "Malaysia", "Germany"}
	locs := []string{"All", "San Diego, California", "Athenry, Ireland", "Bengaluru, India", "Penang, Malaysia", "Remote"}
	tuiModel := tui.NewModel(cats, countries, locs)
	p := tea.NewProgram(tuiModel)
	finalModel, err := p.Run()
	if err != nil {
		log.Fatalf("TUI Error: %v", err)
	}

	m := finalModel.(tui.Model)
	if m.Quitting || !m.Submitted {
		fmt.Println("Selection cancelled.")
		return
	}

	fmt.Printf("Database initialized at %s\n", dbPath)

	// Fetch targets from DB
	targets, err := store.GetTargets()
	if err != nil {
		log.Fatalf("Failed to fetch target companies: %v", err)
	}

	// Apply TUI filters to targets
	for i := range targets {
		targets[i].Category = m.SelectedCat
		targets[i].Country = m.SelectedCountry
		targets[i].Location = m.SelectedLoc
	}

	// Initialize Telemetry
	ollama := telemetry.NewOllamaClient("", "llama3")
	vault := telemetry.NewVaultWriter("")

	// 1. Scrape Concurrently
	fmt.Println("Triggering concurrent scraper...")
	huntScraper := scraper.NewScraper(3)
	results := huntScraper.Run(targets)

	// 2. Queue for Sequential AI Processing
	analysisChan := make(chan PipelineJob, 100)
	var wg sync.WaitGroup

	// Start sequential AI worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		for pJob := range analysisChan {
			// Check if new
			isNew, err := store.IsJobNew(pJob.Job.JobID)
			if err != nil {
				log.Printf("DB error checking job %s: %v", pJob.Job.JobID, err)
				continue
			}
			if !isNew {
				continue
			}

			fmt.Printf("Analyzing new job: %s - %s\n", pJob.Company.Name, pJob.Job.Title)

			// Analyze (Ollama)
			// Note: In a real scenario, we'd fetch the full description here.
			// For now, we pass the title as a placeholder for description.
			analysis, err := ollama.AnalyzeJob(pJob.Job.Title)
			if err != nil {
				log.Printf("AI analysis failed for job %s (falling back to empty analysis): %v", pJob.Job.JobID, err)
				analysis = &telemetry.AnalysisResult{
					RoleType: "Unknown",
				}
			}

			// Save to DB
			if err := store.SaveJob(pJob.Company.Name, pJob.Job, analysis); err != nil {
				log.Printf("DB error saving job %s: %v", pJob.Job.JobID, err)
				continue
			}

			// Export to Vault
			if err := vault.WriteJob(pJob.Company.Name, pJob.Job, analysis); err != nil {
				log.Printf("Vault error exporting job %s: %v", pJob.Job.JobID, err)
				continue
			}
		}
	}()

	// Feed results into the analysis queue
	for _, res := range results {
		if res.Error != nil {
			fmt.Printf("Error scraping %s: %v\n", res.CompanyName, res.Error)
			continue
		}

		// Find the target company for context
		var target scraper.TargetCompany
		for _, t := range targets {
			if t.Name == res.CompanyName {
				target = t
				break
			}
		}

		for _, job := range res.Jobs {
			analysisChan <- PipelineJob{
				Company: target,
				Job:     job,
			}
		}
	}
	close(analysisChan)
	wg.Wait()

	fmt.Println("Hunting completed.")
}

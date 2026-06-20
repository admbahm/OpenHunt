package main

import (
	"fmt"
	"log"

	"github.com/openhunt/openhunt/internal/db"
	"github.com/openhunt/openhunt/internal/scraper"
)

func main() {
	fmt.Println("Starting openHunt...")

	// Initialize the database
	dbPath := "database/openhunt.db"
	store, err := db.NewSQLStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer store.Close()

	fmt.Printf("Database initialized at %s\n", dbPath)

	// Mock target companies
	targets := []scraper.TargetCompany{
		{
			Name:    "Illumina",
			Tenant:  "illumina",
			Site:    "illumina_external",
			BaseURL: "https://illumina.wd3.myworkdayjobs.com/illumina_external",
		},
		{
			Name:    "Dexcom",
			Tenant:  "dexcom",
			Site:    "dexcom_external",
			BaseURL: "https://dexcom.wd3.myworkdayjobs.com/dexcom_external",
		},
	}

	// Initialize and run the scraper
	fmt.Println("Triggering concurrent scraper test run...")
	huntScraper := scraper.NewScraper(3) // 3 concurrent workers
	results := huntScraper.Run(targets)

	// Print results summary
	for _, res := range results {
		if res.Error != nil {
			fmt.Printf("Error scraping %s: %v\n", res.CompanyName, res.Error)
			continue
		}
		fmt.Printf("Discovered %d jobs at %s\n", len(res.Jobs), res.CompanyName)
	}

	fmt.Println("Ready to hunt.")
}

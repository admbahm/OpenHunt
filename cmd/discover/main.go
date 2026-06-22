package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/openhunt/openhunt/internal/db"
	"github.com/openhunt/openhunt/internal/discovery"
)

func main() {
	dbPath := flag.String("db", "database/openhunt.db", "Path to SQLite database")
	autoSave := flag.Bool("y", false, "Auto-save without prompt")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: go run cmd/discover/main.go [options] \"<Company Name>\"")
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	companyName := args[0]
	fmt.Printf("Searching for %q job board...\n", companyName)

	target, err := discovery.SearchCompanyCareers(companyName)
	if err != nil {
		log.Fatalf("Discovery failed: %v", err)
	}

	fmt.Println("\n--- Discovered Target Details ---")
	fmt.Printf("Company Name : %s\n", target.Name)
	fmt.Printf("Platform     : %s\n", target.Platform)
	fmt.Printf("Tenant       : %s\n", target.Tenant)
	if target.Platform == "workday" {
		fmt.Printf("Site         : %s\n", target.Site)
		fmt.Printf("Base URL     : %s\n", target.BaseURL)
	}
	fmt.Println("---------------------------------")

	// Initialize the database
	store, err := db.NewSQLStore(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer store.Close()

	// Handle saving
	save := *autoSave
	if !save {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\nDo you want to save this to your target companies list? (y/N): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read input: %v", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "y" || input == "yes" {
			save = true
		}
	}

	if save {
		err := store.AddTarget(*target)
		if err != nil {
			log.Fatalf("Failed to save target to database: %v", err)
		}
		fmt.Printf("Successfully added %s to database at %s!\n", target.Name, *dbPath)
	} else {
		fmt.Println("Cancelled saving target company.")
	}
}

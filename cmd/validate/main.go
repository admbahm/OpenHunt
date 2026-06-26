package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openhunt/openhunt/internal/db"
	"github.com/openhunt/openhunt/internal/discovery"
	"github.com/openhunt/openhunt/internal/scraper"
	"github.com/openhunt/openhunt/internal/validation"
)

func main() {
	dbPath := flag.String("db", "database/openhunt.db", "Path to SQLite database")
	company := flag.String("company", "", "Validate only one company by name")
	platform := flag.String("platform", "", "Validate only one ATS platform")
	category := flag.String("category", "All", "Runtime category filter")
	country := flag.String("country", "All", "Runtime country filter")
	location := flag.String("location", "All", "Runtime location filter")
	timeout := flag.Duration("timeout", 30*time.Second, "HTTP timeout per request")
	jsonOutput := flag.Bool("json", false, "Print machine-readable JSON")
	flag.Parse()
	companyName := normalizeCompanyFlag(*company, flag.Args())

	store, err := db.NewSQLStore(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer store.Close()

	if err := store.SeedTargets(); err != nil {
		log.Fatalf("Failed to seed target companies: %v", err)
	}

	targets, err := store.GetTargets()
	if err != nil {
		log.Fatalf("Failed to fetch target companies: %v", err)
	}

	targets = filterTargets(targets, companyName, *platform)
	if len(targets) == 0 {
		discovered, err := discoverUnconfiguredTarget(companyName, *platform)
		if err != nil {
			if *jsonOutput {
				printJSON([]validation.Result{discoveryFailureResult(companyName, *platform, err)})
			} else {
				printDiscoveryFailure(companyName, err, store)
			}
			os.Exit(1)
		}
		targets = []scraper.TargetCompany{*discovered}
	}

	for i := range targets {
		targets[i].Category = *category
		targets[i].Country = *country
		targets[i].Location = *location
	}

	validator := validation.Validator{
		HTTPClient: &http.Client{Timeout: *timeout},
	}
	results := validator.ValidateAll(targets)

	if *jsonOutput {
		printJSON(results)
	} else {
		printReport(results)
	}

	if hasFailures(results) {
		os.Exit(1)
	}
}

func normalizeCompanyFlag(company string, args []string) string {
	parts := []string{}
	if strings.TrimSpace(company) != "" {
		parts = append(parts, strings.TrimSpace(company))
	}
	for _, arg := range args {
		if strings.TrimSpace(arg) != "" {
			parts = append(parts, strings.TrimSpace(arg))
		}
	}
	return strings.Join(parts, " ")
}

func filterTargets(targets []scraper.TargetCompany, company, platform string) []scraper.TargetCompany {
	company = strings.ToLower(strings.TrimSpace(company))
	platform = strings.ToLower(strings.TrimSpace(platform))

	filtered := make([]scraper.TargetCompany, 0, len(targets))
	for _, target := range targets {
		if company != "" && !strings.Contains(strings.ToLower(target.Name), company) {
			continue
		}
		if platform != "" && strings.ToLower(target.Platform) != platform {
			continue
		}
		filtered = append(filtered, target)
	}
	return filtered
}

func discoverUnconfiguredTarget(company, platform string) (*scraper.TargetCompany, error) {
	if strings.TrimSpace(company) == "" {
		return nil, fmt.Errorf("no target companies matched the requested filters")
	}
	target, err := discovery.SearchCompanyCareers(company)
	if err != nil {
		return nil, err
	}
	if platform != "" && !strings.EqualFold(target.Platform, platform) {
		return nil, fmt.Errorf("discovered %s target, but --platform requested %s", target.Platform, platform)
	}
	return target, nil
}

func discoveryFailureResult(company, platform string, err error) validation.Result {
	if strings.TrimSpace(company) == "" {
		company = "<unspecified>"
	}
	return validation.Result{
		Company:  company,
		Platform: platform,
		OK:       false,
		Errors:   []string{err.Error()},
	}
}

func printJSON(results []validation.Result) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		log.Fatalf("Failed to encode validation results: %v", err)
	}
}

func printDiscoveryFailure(company string, err error, store *db.SQLStore) {
	fmt.Printf("No configured target matched %q.\n", company)

	var unsupported *discovery.UnsupportedATSError
	if errors.As(err, &unsupported) {
		fmt.Printf("Detected unsupported ATS: %s\n", unsupported.ATS)
		if unsupported.URL != "" {
			fmt.Printf("URL: %s\n", unsupported.URL)
		}
		fmt.Println()
		fmt.Printf("openHunt does not have a %q scraper yet. Add support for this ATS, or use cmd/ingest for manual listings from this site.\n", unsupported.ATS)
		return
	}

	fmt.Printf("Discovery also failed: %v\n\n", err)
	printKnownTargets(store)
}

func printKnownTargets(store *db.SQLStore) {
	targets, err := store.GetTargets()
	if err != nil || len(targets) == 0 {
		return
	}
	fmt.Println("Configured targets:")
	for _, target := range targets {
		fmt.Printf("  - %s (%s)\n", target.Name, target.Platform)
	}
}

func hasFailures(results []validation.Result) bool {
	for _, result := range results {
		if !result.OK {
			return true
		}
	}
	return false
}

func printReport(results []validation.Result) {
	fmt.Println("openHunt target validation")
	fmt.Println()
	fmt.Printf("%-24s %-11s %-8s %-9s %-13s %s\n", "Company", "Platform", "Jobs", "Desc", "Duration", "Status")
	fmt.Printf("%-24s %-11s %-8s %-9s %-13s %s\n", strings.Repeat("-", 24), strings.Repeat("-", 11), strings.Repeat("-", 8), strings.Repeat("-", 9), strings.Repeat("-", 13), strings.Repeat("-", 20))

	for _, result := range results {
		status := "PASS"
		if !result.OK {
			status = "FAIL"
		} else if len(result.Warnings) > 0 {
			status = "WARN"
		}

		fmt.Printf("%-24s %-11s %-8d %-9d %-13s %s\n",
			truncate(result.Company, 24),
			result.Platform,
			result.JobCount,
			result.DescriptionCount,
			result.Duration.Round(time.Millisecond),
			status,
		)

		fmt.Printf("  target: tenant=%s", result.Tenant)
		if result.Site != "" {
			fmt.Printf(" site=%s", result.Site)
		}
		if result.BaseURL != "" {
			fmt.Printf(" base_url=%s", result.BaseURL)
		}
		fmt.Println()

		for _, errMsg := range result.Errors {
			fmt.Printf("  error: %s\n", errMsg)
		}
		for _, warning := range result.Warnings {
			fmt.Printf("  warning: %s\n", warning)
		}
		for _, sample := range result.SampleJobs {
			fmt.Printf("  sample: %s | %s | %s\n", sample.ID, sample.Title, sample.Location)
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "."
}

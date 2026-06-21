package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/openhunt/openhunt/internal/db"
	"github.com/openhunt/openhunt/internal/scraper"
	"github.com/openhunt/openhunt/internal/telemetry"
)

func main() {
	companyFlag := flag.String("company", "Illumina", "Company name")
	titleFlag := flag.String("title", "", "Job title")
	locationFlag := flag.String("location", "", "Job location")
	postedFlag := flag.String("posted", "", "Posted date string")
	urlFlag := flag.String("url", "", "Job posting URL")
	idFlag := flag.String("id", "", "Job Requisition ID")
	dbPathFlag := flag.String("db", "database/openhunt.db", "Path to SQLite database")
	vaultPathFlag := flag.String("vault", "Market-Insights", "Path to Obsidian vault")
	descFileFlag := flag.String("desc-file", "", "Path to a file containing the raw job description (optional)")

	flag.Parse()

	// Gather job description
	var description string
	if *descFileFlag != "" {
		bytes, err := os.ReadFile(*descFileFlag)
		if err != nil {
			log.Fatalf("Failed to read description file: %v", err)
		}
		description = string(bytes)
	} else {
		// Read from stdin if no file is provided
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			var lines []string
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			description = strings.Join(lines, "\n")
		}
	}

	if description == "" {
		log.Println("Warning: Job description is empty. Run with description piped or via -desc-file.")
	}

	if *titleFlag == "" {
		log.Fatal("-title is required")
	}
	if *idFlag == "" {
		log.Fatal("-id is required")
	}

	// Initialize the database
	store, err := db.NewSQLStore(*dbPathFlag)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer store.Close()

	// Initialize Vault
	vault := telemetry.NewVaultWriter(*vaultPathFlag)

	// Build job model
	job := scraper.JobListing{
		JobID:         *idFlag,
		Title:         *titleFlag,
		LocationsText: *locationFlag,
		PostedOn:      *postedFlag,
		ExternalPath:  *urlFlag,
	}

	// Try Ollama analysis
	ollama := telemetry.NewOllamaClient("", "llama3")
	var analysis *telemetry.AnalysisResult
	analysis, err = ollama.AnalyzeJob(description)
	if err != nil {
		log.Printf("Ollama analysis failed (falling back to smart heuristic parsing): %v", err)
		analysis = runSmartHeuristics(description, *titleFlag)
	}

	// Save to DB
	if err := store.SaveJob(*companyFlag, job, analysis); err != nil {
		log.Fatalf("Failed to save job to database: %v", err)
	}
	fmt.Printf("Successfully saved job %s to database.\n", job.JobID)

	// Save to Vault
	if err := vault.WriteJob(*companyFlag, job, analysis); err != nil {
		log.Fatalf("Failed to export job to Obsidian vault: %v", err)
	}
	fmt.Printf("Successfully exported job to Obsidian vault.\n")

	// Print summary of analysis
	fmt.Printf("\n--- Extracted Intelligence ---\n")
	fmt.Printf("Salary Min: $%d\n", analysis.BaseSalaryMin)
	fmt.Printf("Salary Max: $%d\n", analysis.BaseSalaryMax)
	fmt.Printf("Role Type:  %s\n", analysis.RoleType)
	fmt.Printf("Tech Stack: %s\n", strings.Join(analysis.TechStack, ", "))
	fmt.Printf("Regulatory: %s\n", strings.Join(analysis.RegulatoryGates, ", "))
}

// runSmartHeuristics implements static keyword/regex matching to extract metadata.
func runSmartHeuristics(description, title string) *telemetry.AnalysisResult {
	res := &telemetry.AnalysisResult{
		RoleType: "Individual Contributor",
	}

	// Determine Role Type
	lowerTitle := strings.ToLower(title)
	if strings.Contains(lowerTitle, "manager") || strings.Contains(lowerTitle, "director") || strings.Contains(lowerTitle, "head of") || strings.Contains(lowerTitle, "lead") {
		res.RoleType = "Management"
	}

	// Extract Salary Range
	// Look for pattern like: $129,400 - $194,000 or $129,400 to $194,000
	salaryRegex := regexp.MustCompile(`\$([0-9]{1,3}(?:,[0-9]{3})*)\s*(?:-|to)\s*\$([0-9]{1,3}(?:,[0-9]{3})*)`)
	matches := salaryRegex.FindStringSubmatch(description)
	if len(matches) == 3 {
		minStr := strings.ReplaceAll(matches[1], ",", "")
		maxStr := strings.ReplaceAll(matches[2], ",", "")
		minVal, err1 := strconv.Atoi(minStr)
		maxVal, err2 := strconv.Atoi(maxStr)
		if err1 == nil && err2 == nil {
			res.BaseSalaryMin = minVal
			res.BaseSalaryMax = maxVal
		}
	}

	// Tech Stack heuristics
	techKeywords := []string{
		"x86", "ARM", "RISC-V", "DDR", "LPDDR", "PCIe", "Ethernet", "PHY", "PHYs", "U-Boot", "UEFI", "BIOS",
		"Linux", "kernel", "device tree", "TPM", "TEE", "PCB", "PCA", "FPGA", "C++", "C ", "Python",
		"ftrace", "perf", "ethtool", "tcpdump", "dmesg", "sysfs", "procfs",
	}
	techMap := make(map[string]bool)
	for _, kw := range techKeywords {
		var re *regexp.Regexp
		if kw == "C " {
			re = regexp.MustCompile(`\bC\b`)
		} else {
			re = regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(kw) + `\b`)
		}
		if re.MatchString(description) {
			cleanKw := strings.TrimSpace(kw)
			if cleanKw == "PHYs" || cleanKw == "PHY" {
				cleanKw = "Ethernet PHY"
			}
			techMap[cleanKw] = true
		}
	}
	for tech := range techMap {
		res.TechStack = append(res.TechStack, tech)
	}

	// Regulatory Gates heuristics
	regKeywords := []string{"FDA", "HIPAA", "GxP", "ISO 13485", "Secret Clearance", "CE-IVD"}
	for _, kw := range regKeywords {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(kw) + `\b`)
		if re.MatchString(description) {
			res.RegulatoryGates = append(res.RegulatoryGates, kw)
		}
	}

	return res
}

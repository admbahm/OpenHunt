package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/openhunt/openhunt/internal/scraper"
)

// VaultWriter handles exporting job data to an Obsidian vault.
type VaultWriter struct {
	BaseDir string
}

// NewVaultWriter initializes a new VaultWriter and ensures the base directory exists.
func NewVaultWriter(baseDir string) *VaultWriter {
	if baseDir == "" {
		baseDir = os.Getenv("OPENHUNT_OUTPUT_DIR")
	}
	if baseDir == "" {
		baseDir = "./Market-Insights"
	}

	// Ensure the base directory and @Closed exist under the resolved base path
	activeDir := filepath.Join(baseDir, "@Active")
	closedDir := filepath.Join(baseDir, "@Closed")

	_ = os.MkdirAll(activeDir, 0755)
	_ = os.MkdirAll(closedDir, 0755)

	return &VaultWriter{BaseDir: activeDir}
}

// WriteJob exports a job listing and its analysis to a Markdown file.
func (v *VaultWriter) WriteJob(companyName string, job scraper.JobListing, analysis *AnalysisResult) error {
	// Ensure directory exists
	if err := os.MkdirAll(v.BaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create vault directory: %w", err)
	}

	// Create a clean filename
	filename := fmt.Sprintf("%s - %s.md", companyName, job.Title)
	filename = strings.ReplaceAll(filename, "/", "-")
	filename = strings.ReplaceAll(filename, ":", "-")
	path := filepath.Join(v.BaseDir, filename)

	// Build Markdown content with YAML frontmatter
	content := fmt.Sprintf(`---
job_id: %s
company: %s
title: "%s"
location: "%s"
posted_at: %s
salary_min: %d
salary_max: %d
role_type: "%s"
tech_stack: [%s]
regulatory_gates: [%s]
scraped_at: %s
---

# %s

**Company:** %s
**Location:** %s
**Posted:** %s

## Intelligence Analysis

- **Role Type:** %s
- **Salary Range:** $%d - $%d
- **Tech Stack:** %s
- **Regulatory Gates:** %s

## Description
[Raw description or link to Workday would go here]
`, job.JobID, companyName, job.Title, job.LocationsText, job.PostedOn,
		analysis.BaseSalaryMin, analysis.BaseSalaryMax, analysis.RoleType,
		strings.Join(quoteSlice(analysis.TechStack), ", "),
		strings.Join(quoteSlice(analysis.RegulatoryGates), ", "),
		time.Now().Format("2006-01-02 15:04:05"),
		job.Title, companyName, job.LocationsText, job.PostedOn,
		analysis.RoleType, analysis.BaseSalaryMin, analysis.BaseSalaryMax,
		strings.Join(analysis.TechStack, ", "),
		strings.Join(analysis.RegulatoryGates, ", "),
	)

	// Write atomically using a temp file
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file to final path: %w", err)
	}

	return nil
}

func quoteSlice(s []string) []string {
	quoted := make([]string, len(s))
	for i, v := range s {
		quoted[i] = fmt.Sprintf("\"%s\"", v)
	}
	return quoted
}

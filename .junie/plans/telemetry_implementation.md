# Technical Plan: Local Intelligence Module (Telemetry)

This document outlines the implementation plan for the `internal/telemetry/` module, focusing on local AI processing via Ollama and Obsidian vault export.

## 1. Ollama Client (`internal/telemetry/ollama.go`)

- **Interface**: `IntelligenceProvider`
    - `AnalyzeJob(description string) (*AnalysisResult, error)`
- **HTTP Client**:
    - **Endpoint**: `http://localhost:11434/api/generate` (default)
    - **Timeout**: 30-60 seconds (AI generation is slow).
    - **Payload**:
      ```json
      {
        "model": "llama3",
        "prompt": "...",
        "stream": false,
        "format": "json"
      }
      ```
- **Prompt Engineering**:
    - Instruct the model to extract specific fields from the job description.
    - Enforce JSON output.

## 2. Structured Schema (`internal/telemetry/models.go`)

```go
type AnalysisResult struct {
	BaseSalaryMin   int      `json:"base_salary_min"`
	BaseSalaryMax   int      `json:"base_salary_max"`
	TechStack       []string `json:"tech_stack"`
	RegulatoryGates []string `json:"regulatory_gates"`
	RoleType        string   `json:"role_type"` // e.g., "Individual Contributor", "Management"
}
```

## 3. Vault Export (`internal/telemetry/vault.go`)

- **Utility**: `WriteToVault(job scraper.JobListing, analysis AnalysisResult)`
- **Target Directory**: `Market-Insights/@Active/`
- **Format**: Markdown with YAML frontmatter.
- **Atomicity**: Write to a temporary file and rename to ensure no partial writes.

## 4. Pipeline Orchestration (`internal/pipeline/pipeline.go` or `cmd/openhunt/main.go`)

The core loop logic:
1.  **Scrape**: Get `JobListing`s from `internal/scraper`.
2.  **Filter**: For each job, check `internal/db` if `JobID` already exists.
3.  **Analyze**: If new, fetch full description (may need a new scraper method) and send to `internal/telemetry`.
4.  **Save DB**: Store the job and its analysis result in the SQLite database.
5.  **Save Vault**: Generate the Markdown file in the Obsidian vault.

## 5. Database Schema Updates (`internal/db/db.go`)

- Update `migrate()` to include analysis fields in the `jobs` table or a new `job_analysis` table.
- Add `IsJobNew(jobID string) (bool, error)` to `Store`.
- Add `SaveJob(job scraper.JobListing, analysis *telemetry.AnalysisResult) error` to `Store`.

## 6. Verification Plan

- **Ollama Integration**: Manual check with a running local Ollama instance (or mock the HTTP response for CI-like validation).
- **Markdown Generation**: Verify file content and pathing.
- **Pipeline Logic**: Ensure the "skip if exists" logic works correctly.

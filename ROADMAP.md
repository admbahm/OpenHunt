# openHunt Roadmap

This roadmap outlines the planned enhancements and feature additions to openHunt to build a more robust, comprehensive, and user-friendly job search intelligence engine.

## Core Scraper & Engine Enhancements

### [x] Workday Scraper Pagination
- **Status**: Completed. The scraper follows the Workday response `total`, requests pages in 20-result increments, and preserves all successfully fetched pages.
- **Rate limiting**: A randomized 200–500ms delay is applied between page requests, with per-page progress logging.
- **Reference**: See [PRD](docs/prd_pagination.md) and [Technical Design](docs/tech_design_pagination.md).

### [x] Target Validation & Health Checks
- **Status**: Completed. `cmd/validate` checks configured target companies for supported ATS metadata, scrape reachability, returned job counts, description extraction, scrape duration, warnings, and actionable errors.
- **Usage**: Run `go run cmd/validate/main.go`, optionally filtering with `--company`, `--platform`, `--category`, `--country`, or `--location`.
- **Output**: Prints a concise pass/warn/fail report by default and supports `--json` for machine-readable validation results.

### [ ] Shared HTTP Retry, Backoff, and Typed Errors
- **Goal**: Improve scraper resilience across all supported ATS platforms.
- **Details**: Standardize timeouts, transient-status retries, exponential backoff with jitter, response body limits, and user-agent handling for Workday, Greenhouse, Lever, and Ashby.
- **Typed errors**: Introduce errors such as `ErrRateLimited`, `ErrUnsupportedBoard`, `ErrMalformedResponse`, and `ErrTemporaryFailure` so the CLI, TUI, and logs can distinguish failure modes.

### [ ] Location Normalization Layer
- **Goal**: Standardize fragmented location strings to improve searchability and Dataview querying in Obsidian.
- **Details**: Implement a post-processing normalization layer to map highly fragmented location strings (e.g., `San Diego, California`, `US - California - San Diego`, `3 Locations`, `USA - Remote`) to standard tags (e.g., `San Diego Metro`, `Remote`, `Multi-Location`) before exporting frontmatter.

### [ ] Log Unsupported ATS Metrics to a Backlog
- **Goal**: Capture unsupported ATS targets to prioritize which platforms to implement next.
- **Details**: Write events of `UnsupportedATSError` to an `unsupported_targets.json` file or SQLite database table to track frequency of encountered platforms (e.g. iCIMS, BrassRing).

### [ ] Obsidian Export Sanitization (Fix Output Folder Nesting)
- **Goal**: Prevent output folder nesting issues in the Obsidian export pipeline.
- **Details**: Ensure the output path logic sanitizes trailing subdirectories and does not duplicate output folder structures (e.g. nesting notes into `@Active/@Active` folders like `Market-Insights/@Active/@Active/Adobe - ...`).

### [ ] Stable Job Identity & Duplicate Detection
- **Goal**: Reduce duplicate churn and protect against ATS ID drift or incomplete source data.
- **Details**: Keep ATS IDs as the primary key where reliable, but add a secondary fingerprint derived from platform, tenant, normalized title, normalized location, and normalized apply URL.
- **Use cases**: Detect reposts, changed ATS identifiers, missing IDs, and cross-run duplicates before invoking the AI pipeline.

### [ ] Closed Job Reconciliation
- **Goal**: Track jobs that disappear from source ATS boards and keep the Obsidian vault current.
- **Details**: Compare active job IDs from each scrape against jobs currently marked active in SQLite, mark missing jobs as closed, and move corresponding Markdown notes from `@Active` to `@Closed`.
- **Database changes**: Add job status fields such as `active`, `closed_at`, and `last_seen_at`.

### [ ] CLI Ingestion & Manual Override Improvements
- **Goal**: Make it easier for users to feed manually discovered listings directly into the intelligence pipeline.
- **Details**: Enhance the `cmd/ingest` utility to support interactive editing of extracted metadata before committing to SQLite and Obsidian.

### [ ] Non-Interactive CLI Mode
- **Goal**: Support repeatable automation, scheduled runs, and CI-style checks without requiring the TUI.
- **Details**: Add flags for `--category`, `--country`, `--location`, `--company`, `--platform`, `--no-ai`, `--export-only`, and `--retry-failed`.
- **Behavior**: Preserve the TUI as the default interactive mode while allowing explicit flags to bypass it.

### [ ] File-Based Target Configuration
- **Goal**: Make target company configuration easier to review, edit, and version.
- **Details**: Support a local YAML or TOML target configuration file that can sync into SQLite, replacing the need to edit seed data in Go source for normal use.
- **Validation**: Reuse the target validation command to verify config changes before they are persisted.

### [ ] Proxy and Rate Limit Mitigation
- **Goal**: Protect scraper requests against IP bans, rate limits, and Cloudflare challenges.
- **Details**: Implement user-agent rotation, proxy rotation, and randomized delays (jitter) between paginated queries.

### [ ] Scraper Fixture Test Suite
- **Goal**: Catch ATS response-shape changes and prevent regressions in normalization logic.
- **Details**: Add saved fixture payloads/pages for Workday, Greenhouse, Lever, and Ashby, then test ID normalization, description extraction, location/category filtering, malformed responses, and error classification.

---

## User Interface (UI) Development

### [ ] Configurable TUI Dropdown Filter Options
- **Goal**: Allow users to configure the Category, Country, and Location options displayed in the Bubble Tea TUI.
- **Details**: Support loading these options dynamically from a local configuration file (YAML/TOML) or by querying the unique values populated in the target database.

### [ ] Local Web Dashboard
- **Goal**: Provide a clean, premium graphical user interface to visualize scraped jobs, salary trends, and stack distributions.
- **Tech Stack**: Next.js, Tailwind CSS (configured for modern dark mode aesthetics), and SQLite backend.
- **Key Features**:
  - Live scraper status and log output.
  - Searchable/filterable table of all jobs in the database.
  - Statistics charts showing salary distributions and commonly requested technologies.

---

## AI & Telemetry Improvements

### [ ] Structured Extraction Fallbacks & Verification
- **Goal**: Enhance accuracy and reliability of intelligence extraction even when Ollama is unavailable or misbehaving.
- **Details**: Integrate a regex and heuristic fallback engine directly into the main pipeline. Extract salary ranges, role type, tech keywords, and regulatory gates before or after Ollama enrichment.
- **Verification**: Add validation steps to assert that salary ranges, role classifications, and extracted lists are plausible before saving them.

### [ ] Structured JSON Output from Ollama
- **Goal**: Eliminate parsing errors by forcing structured JSON output from the local LLM.
- **Details**: Leverage Ollama's structured JSON format support (e.g. combining the `format: "json"` option with a specific schema structure/prompt formatting) to prevent corrupt job notes caused by malformed frontmatter syntax or invalid Markdown.

### [ ] Pipeline State Tracking & Retryable Exports
- **Goal**: Prevent SQLite records and Obsidian exports from drifting when one stage succeeds and another fails.
- **Details**: Track `analysis_status`, `vault_export_status`, `last_error`, and `updated_at` on jobs so failed analysis or export steps can be retried without re-scraping.
- **Commands**: Add a retry path for failed analysis/export jobs.

### [ ] Per-ATS Scrape Telemetry
- **Goal**: Detect source instability and ATS-specific breakage quickly.
- **Details**: Record last successful scrape, last error, returned job count, HTTP status, duration, and endpoint/platform metadata for each target company.
- **Use cases**: Surface stale targets, repeated rate limits, empty scrape results, and platform-specific failures in both logs and future dashboards.

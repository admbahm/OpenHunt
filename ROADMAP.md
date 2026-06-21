# openHunt Roadmap

This roadmap outlines the planned enhancements and feature additions to openHunt to build a more robust, comprehensive, and user-friendly job search intelligence engine.

## Core Scraper & Engine Enhancements

### [x] Workday Scraper Pagination
- **Status**: Completed. The scraper follows the Workday response `total`, requests pages in 20-result increments, and preserves all successfully fetched pages.
- **Rate limiting**: A randomized 200–500ms delay is applied between page requests, with per-page progress logging.
- **Reference**: See [PRD](docs/prd_pagination.md) and [Technical Design](docs/tech_design_pagination.md).

### [ ] CLI Ingestion & Manual Override Improvements
- **Goal**: Make it easier for users to feed manually discovered listings directly into the intelligence pipeline.
- **Details**: Enhance the `cmd/ingest` utility to support interactive editing of extracted metadata before committing to SQLite and Obsidian.

### [ ] Proxy and Rate Limit Mitigation
- **Goal**: Protect scraper requests against IP bans, rate limits, and Cloudflare challenges.
- **Details**: Implement user-agent rotation, proxy rotation, and randomized delays (jitter) between paginated queries.

---

## User Interface (UI) Development

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
- **Details**: Integrate the smart regex-based fallback engine directly into the main pipeline. Add verification steps to assert that salary ranges and role classifications are parsed correctly.

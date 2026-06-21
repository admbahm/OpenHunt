# OpenHunt Handoff Document

> **Last Updated**: 2026-06-21

## Project Overview
OpenHunt is a job board intelligence engine that scrapes listings from multiple ATS platforms (Workday, Greenhouse), deduplicates them against a local SQLite database, and runs AI-powered analysis via Ollama to extract tech stacks, salary ranges, and regulatory requirements. Results are exported as structured Obsidian vault markdown.

## Current State — Stable ✅

The project is in a **hardened, pre-feature-expansion state**. Both scraper backends are functional, the test suite is green, and the core pipeline runs end-to-end.

### What's Working
- **Workday Scraper** (`internal/scraper/client.go`):
  - Stateful session management: GET-based token/cookie harvesting → CSRF injection on POST.
  - Dynamic facet resolution with 4-pass matching (exact → partial, scoped → global) across nested facet trees.
  - `locationMainGroup` remapping to correct sub-parameters (`locationCountry`, `locations`, `locationHierarchy1`).
  - **Full pagination**: Loops over all pages in 20-result increments using `total` from the API response. Jittered 200–500ms backoff between pages. Operational logging per page: `[CompanyName] Fetched page offset N/T...`.
  - Duplicate facet ID deduplication when country and location resolve to the same parameter.

- **Greenhouse Scraper** (`internal/scraper/greenhouse.go`):
  - Public boards API integration (`/v1/boards/{tenant}/jobs?content=true`).
  - TUI numeric prefix stripping (regex: `^\d+\.\s*`) for all runtime filter fields.
  - **Hardened location filter**: Any filter string *containing* "remote" (case-insensitive) triggers a relaxed `strings.Contains` check for `"remote"` in the job location, rather than requiring an exact token match. Fixes the edge case where TUI passes values like `"6. Remote"` or `"Remote, US"`.
  - Loose category matching via case-insensitive substring against department names.

- **Architecture**:
  - Factory pattern (`NewScraperFactory`) dispatches to the correct backend by platform string.
  - `JobScraper` interface ensures all backends share a single `FetchJobs(TargetCompany)` contract.
  - Concurrent worker pool in `scraper.go` for multi-target parallel execution.
  - Custom `UnmarshalJSON` on `JobListing` handles Workday's `bulletinNumber`/`bulletFields` ID extraction.

- **Database** (`internal/db`): SQLite via CGO-free `ncruces/go-sqlite3/driver`. `jobs` table for listings, `target_companies` for seed configs.
- **TUI** (`internal/tui`): Bubble Tea dashboard with Category, Country, and Location filter selectors.
- **AI Pipeline** (`internal/telemetry`): Ollama integration for metadata extraction, with Obsidian vault sync to `Market-Insights/@Active` and `@Closed`.
- **CLI Tools**: `cmd/openhunt` (main orchestration), `cmd/ingest` (manual listing ingestion), `cmd/debug_facets` and `cmd/debug_greenhouse` (diagnostic utilities).

### Test Suite
```
ok   internal/db        — passes
ok   internal/scraper   — 7 tests pass (factory, Greenhouse fetch, loose filtering, facet resolution, locationMainGroup mapping, request structure, orchestrator)
```

## Resolved Blockers
The following issues from previous sessions are **resolved**:

| Issue | Resolution |
|---|---|
| Workday HTTP 400 on filtered queries | Fixed via dynamic facet key discovery and `locationMainGroup` → sub-parameter remapping |
| Workday returning only first 20 results | Implemented full pagination loop with `offset`/`total` tracking |
| Greenhouse "remote" filter missing `"Remote, US"` jobs | Changed filter from exact `== "remote"` to `strings.Contains` on any filter mentioning "remote" |
| TUI numeric prefixes leaking into API queries | `stripNumericPrefix` regex applied to all three filter fields before any matching logic |

## What's Next
Refer to `ROADMAP.md` for the full feature backlog. Immediate priorities:

1. **State Layout Changes** — The scraper core is now hardened; the next phase involves restructuring the TUI state layouts (deferred until accuracy work was complete).
2. **CLI Ingestion Improvements** — Interactive editing of extracted metadata before commit.
3. **Proxy & Rate Limit Mitigation** — User-agent rotation, proxy support, and expanded jitter strategies.
4. **Local Web Dashboard** — Next.js + Tailwind dark-mode UI for visualizing scraped data, salary trends, and stack distributions.
5. **AI Fallback Hardening** — Regex/keyword extraction fallback when Ollama is unavailable; structured verification of salary and role classification outputs.

## Key Files
| File | Purpose |
|---|---|
| `internal/scraper/client.go` | Workday CXS scraper — session management, facet resolution, paginated fetching |
| `internal/scraper/greenhouse.go` | Greenhouse public API scraper with hardened location/category filtering |
| `internal/scraper/models.go` | Shared types: `TargetCompany`, `JobListing`, `WorkdayRequest/Response` |
| `internal/scraper/factory.go` | Platform-based scraper instantiation |
| `internal/scraper/scraper.go` | Concurrent worker pool orchestrator |
| `internal/db/db.go` | SQLite schema, target seeding, deduplication |
| `internal/tui/dashboard.go` | Bubble Tea filter selection UI |
| `internal/telemetry/` | Ollama AI analysis and Obsidian vault export |
| `cmd/openhunt/main.go` | Main entry point and pipeline orchestration |
| `cmd/ingest/` | Manual listing ingestion CLI |
| `docs/` | PRDs, architecture doc, technical designs |

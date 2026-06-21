# Workspace Rules for openHunt

This document contains style guidelines, architecture rules, and development constraints specific to the `openHunt` workspace. All agents working on this project must adhere to these guidelines.

## Code & Architecture Guidelines

### Go Development
- Use Go 1.25+ features and patterns.
- Do not introduce CGO dependencies. We use the CGO-free `github.com/ncruces/go-sqlite3/driver` for SQLite interaction.
- Keep components structured and decoupled:
  - `internal/scraper` handles ATS vendor connections.
  - `internal/db` handles local state and deduplication.
  - `internal/telemetry` handles AI processing and Obsidian vault synchronization.

### Web Scraping Integrity
- **Session State**: Always perform stateful token harvesting (GET request to the landing page first) before attempting API requests. This extracts the CSRF token and registers the cookies.
- **Pagination**: When scraping endpoints that support pagination, always query the API incrementally and parse the total count to avoid missing listings.
- **Polite Scraping**: Introduce jittered backoffs or short delays (e.g., 200–500ms) between page requests to avoid hitting rate limits.

### AI Processing & Fallbacks
- Always maintain robust fallback mechanisms for AI metadata extraction (like regex/keyword matching in Go) so that the application runs gracefully if Ollama is not active or available.
- Frontmatter output in Obsidian vault markdown must strictly match YAML expectations.

# Architecture Overview

`openHunt` is designed as a sovereign, local-first market intelligence engine. It works by targeting Workday CXS endpoints, persisting new listings to a local SQLite database, analyzing them using a local Ollama LLM, and exporting structured findings to an Obsidian vault.

## System Layout

```mermaid
graph TD
    A[Scraper Runner] -->|Target Config| B(Stateful Workday Client)
    B -->|1. GET base URL| C[Harvest Cookie / CSRF]
    B -->|2. POST JSON search| D[Retrieve Job Listings]
    D -->|Job Payload| E[SQLite Diff Engine]
    E -->|Is job new?| F{New Job?}
    F -->|Yes| G[Sequential AI worker]
    F -->|No| H[Discard]
    G -->|Prompt + Title| I[Ollama Client]
    I -->|JSON analysis| J[Obsidian Vault Writer]
    J -->|Atomic Markdown file| K[Obsidian Vault]
```

## 1. Stateful Scraping client (`internal/scraper`)

Workday job boards require a stateful session to prevent CSRF exploits and verify web browser authenticity. The scraper client uses a Go `cookiejar` and splits execution into two phases:

1. **GET Request (Token Harvesting)**: A `GET` request is made to the main public-facing landing page of the career portal (e.g. `https://illumina.wd1.myworkdayjobs.com/en-US/illumina-careers/`). This forces Workday to initialize the session and drop the `CALYPSO_CSRF_TOKEN` cookie.
2. **POST Request (JSON query)**: A `POST` request containing a specific JSON payload is made to the internal API endpoint (e.g. `https://{tenant}.wd1.myworkdayjobs.com/wday/cxs/{tenant}/{site}/jobs`). The CSRF token is extracted from the cookie jar and set in the `X-Calypso-Csrf-Token` header.

## 2. SQLite Diff Engine (`internal/db`)

To avoid querying the LLM for identical, previously processed jobs, a local SQLite database (`database/openhunt.db`) is used as a deduplication layer:
- The database stores job IDs, metadata, and the cached AI analysis results.
- Incoming scraped jobs are run through `IsJobNew(jobID)`. If a job ID already exists, it is bypassed, saving computing resources.

## 3. Sequential AI Worker & Ollama (`internal/telemetry`)

Jobs that pass the diff engine are queued sequentially for local LLM processing:
- A worker pulls jobs from the queue and sends their titles/descriptions to a local Ollama server running a model (e.g. `llama2:latest`).
- Ollama is configured to output structured JSON mapping fields like salary range (`base_salary_min`/`base_salary_max`), `tech_stack`, `regulatory_gates`, and `role_type`.

## 4. Obsidian Vault Export (`internal/telemetry`)

The final analysis is written to an Obsidian vault (`Market-Insights/`) as atomic Markdown files:
- The files contain frontmatter conforming to YAML specifications.
- This allows job seekers to search, filter, and link intelligence details directly within their Obsidian workspaces.

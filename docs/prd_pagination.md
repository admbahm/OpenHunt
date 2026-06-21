# PRD: Workday Scraper Pagination

## 1. Objective
Ensure that `openHunt` gathers a complete and comprehensive dataset of all active job listings from targeted Workday career portals. The engine must not miss any opportunities due to arbitrary result limit caps or lack of pagination support.

## 2. Background & User Friction
The current Workday client fetches only the first page of results (up to 20 jobs) starting at offset 0. For high-volume companies (e.g., Illumina, Dexcom), active postings that are older or lower-ranked are pushed off the front page. This results in critical missed job opportunities, defeating the core objective of the sovereign job search assistant.

## 3. Requirements

### 3.1 Pagination Loop
- **P1**: The engine MUST parse the `total` field from the Workday JSON response.
- **P1**: The engine MUST iterate through results, incrementing the `offset` parameter by the pagination `limit` size on each iteration.
- **P1**: The loop MUST terminate when all matching jobs have been retrieved (`retrieved >= total`) or when a page returns zero job postings.
- **P2**: Increase the default page size (`limit`) from `20` to `100` to minimize the number of API requests.

### 3.2 Resilience and Performance
- **P1**: If a request to a single page fails, it should fail gracefully by preserving the already successfully scraped jobs from prior pages, logging the error, and continuing to the next company.
- **P2**: Introduce a minor delay (jittered backoff) between page requests to avoid triggering rate limits on Workday endpoints.

## 4. User Experience (Obsidian / DB)
- No duplicate entries: The existing SQLite unique constraint and `IsJobNew` checks will ensure that paginated results do not produce duplicates.
- All historical jobs matching target criteria should now appear under the active vault directory.

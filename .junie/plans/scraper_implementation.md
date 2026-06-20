# Technical Plan: Concurrent Scraper Module

This document outlines the implementation plan for the `internal/scraper/` module, focusing on concurrent scraping of Workday CXS endpoints.

## 1. Data Structures (`internal/scraper/models.go`)

### `TargetCompany`
Represents a company to be scraped.
- `Name`: string
- `Tenant`: string (extracted from Workday URL)
- `Site`: string (extracted from Workday URL)
- `BaseURL`: string (the original Workday URL for reference)

### `JobListing`
Captures the parsed JSON response from Workday.
- `Title`: string
- `ExternalPath`: string (used to build the full job URL)
- `LocationsText`: string
- `PostedOn`: string
- `JobID`: string

## 2. HTTP Client (`internal/scraper/client.go`)

The `Client` wrapper will handle:
- **Timeouts**: 10-15 second timeout per request.
- **Headers**: 
    - `User-Agent`: A modern browser string.
    - `Content-Type`: `application/json`.
    - `Accept`: `application/json`.
- **Workday CXS API**:
    - **Endpoint**: `https://{tenant}.wd3.myworkdayjobs.com/wday/cxs/{tenant}/{site}/jobs`
    - **Method**: `POST`
    - **Payload**:
      ```json
      {
        "appliedFacets": {},
        "limit": 20,
        "offset": 0,
        "searchText": ""
      }
      ```

## 3. Concurrency Pattern (`internal/scraper/scraper.go`)

We will use a **Worker Pool** pattern:

- **Orchestrator**:
    - Takes `[]TargetCompany`.
    - Creates a `jobs` channel to feed `TargetCompany` to workers.
    - Creates a `results` channel to collect `[]JobListing`.
    - Uses a `sync.WaitGroup` to wait for all workers to finish.
    - Configurable `workerCount` (default 3-5).

- **Worker**:
    - Listens on the `jobs` channel.
    - For each company, calls the HTTP Client.
    - Sends the results to the `results` channel.

- **Graceful Termination**:
    - Close the `jobs` channel when all companies are queued.
    - Wait for workers via `WaitGroup`.
    - Close the `results` channel once workers are done.

## 4. Integration (`cmd/openhunt/main.go`)

- Mock data for `Illumina` and `Dexcom`.
- Initialize `scraper.Scraper`.
- Run the scrape and output the count of jobs found per company.

## 5. Verification Plan

- Since `go` is not available in the environment, verification will be done via:
    - Detailed code review of the concurrency logic (channels, waitgroups).
    - Checking struct tags for JSON mapping.
    - Ensuring error handling is robust (client-side and worker-side).

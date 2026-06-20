# Architectural Plan: openHunt Core Structure

This document outlines the proposed architectural layout and delivery stages for the openHunt project.

## 1. Project Layout & Modularity

We will follow a standard Go project layout to ensure scalability and maintainability.

### Directory Structure
- `cmd/openhunt/`: Entry point for the application. Contains `main.go`.
- `internal/`: Private library code that shouldn't be imported by other projects.
    - `internal/db/`: Local SQLite management, schema migrations, and data access layer.
    - `internal/scraper/`: Concurrent HTTP client for Workday CXS API endpoints.
    - `internal/telemetry/`: Local Ollama instance integration for data processing/analysis.

## 2. Dependencies & Tooling

- **Module Name**: `github.com/openhunt/openhunt`
- **Go Version**: 1.25 (or current stable)
- **Database Driver**: `ncruces/go-sqlite3` (CGO-free SQLite driver)
- **Standard Library**: `database/sql` for abstraction.

## 3. Database Component Design

The database component will be designed with a clear separation between the interface and its implementation.

### Interface (`internal/db/db.go`)
```go
type Store interface {
    Close() error
    // Future methods for data access will be added here
}
```

### Initialization & Migrations
- The system will use a local SQLite file located at `database/openhunt.db`.
- On startup, the application will:
    1. Check if the `database/` directory and `openhunt.db` file exist.
    2. Initialize the file if missing.
    3. Run a migration function that applies schema rules defined in the code or embedded SQL files.

## 4. Delivery Stages

### Stage 1: Foundation (Current)
- Initialize `go.mod` with the correct module path.
- Create the directory skeleton.
- Set up a minimal `cmd/openhunt/main.go` that initializes logging and the database.

### Stage 2: Database Layer
- Implement the `ncruces/go-sqlite3` wrapper.
- Implement the migration logic and initial schema (tables for jobs, scraping status, etc.).

### Stage 3: Scraper Module
- Implement the concurrent HTTP client.
- Define models for Workday CXS API responses.

### Stage 4: Telemetry Module
- Implement the Ollama client for local HTTP requests.
- Integrate the scraping pipeline with the telemetry processing.

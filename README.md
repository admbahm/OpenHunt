# openHunt

openHunt is a sovereign, local-first market intelligence engine designed to counteract asymmetric automated screening pipelines. It empowers job seekers by automating the collection and analysis of job market data using a private, local-first architecture.

## Project Overview

In an era of automated HR filters and asymmetric information, openHunt provides the tools to build your own market intelligence. It targets Workday CXS and Greenhouse job-board APIs, processes the data through a local SQLite diff engine, and leverages local LLMs (via Ollama) to extract structured insights without leaking data to third-party providers.

## Architecture

The system operates as a multi-stage pipeline:

1.  **Concurrent Scraper**: A high-performance Go worker pool dispatches each target to its Workday or Greenhouse backend and processes multiple companies simultaneously.
2.  **SQLite Diff Engine**: A local database layer that identifies new job listings by comparing incoming data against historical records, ensuring only fresh insights are processed.
3.  **Sequential AI Pipeline**: A single-threaded queue that passes new job descriptions to a local **Ollama** instance. This stage extracts structured data (salary ranges, tech stack, regulatory requirements) using models like `llama3`.
4.  **Obsidian Vault Export**: The final intelligence is exported as atomic Markdown files with YAML frontmatter, ready for deep analysis and indexing within an **Obsidian** vault.

## Tech Stack

-   **Go**: The core engine, providing high-performance concurrency and type safety.
-   **SQLite**: A lightweight, CGO-free local database for persistence and deduplication.
-   **Ollama**: Powering local intelligence for structured data extraction from raw text.
-   **Markdown/Obsidian**: The delivery layer for human-readable, searchable market insights.

## Documentation

For more detailed guides, see:
- [Architecture Overview](docs/architecture.md)
- [Configuration Guide](docs/configuration.md)

## Getting Started

### Prerequisites

- Go 1.25+
- Ollama installed and running with an active model (e.g. `llama3` or `llama2`).
- SQLite (handled automatically by the Go driver).

### Installation & Run

1. **Clone the repository**:
   ```bash
   git clone https://github.com/openhunt/openhunt.git
   cd openhunt
   ```

2. **Setup directories**:
   The application will automatically create the `database/` and `Market-Insights/` directories on first run.

3. **Run the engine**:
   You can optionally configure the Ollama endpoint and model using environment variables:
   ```bash
   OLLAMA_API_URL="http://localhost:11434" OLLAMA_MODEL="llama3" go run cmd/openhunt/main.go
   ```

4. **Analyze the results**:
   Open the `Market-Insights/` folder in Obsidian to view your structured market intelligence.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

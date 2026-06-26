```text
  ___                   _   _             _
 / _ \ _ __   ___ _ __ | | | |_   _ _ __ | |_
| | | | '_ \ / _ \ '_ \| |_| | | | | '_ \| __|
| |_| | |_) |  __/ | | |  _  | |_| | | | | |_
 \___/| .__/ \___|_| |_|_| |_|\__,_|_| |_|\__|
      |_|
```

# openHunt

openHunt is a sovereign, local-first market intelligence engine designed to counteract asymmetric automated screening pipelines. It empowers job seekers by automating the collection and analysis of job market data using a private, local-first architecture.

## Project Overview

In an era of automated HR filters and asymmetric information, openHunt provides the tools to build your own market intelligence. It targets Workday CXS, Greenhouse, Lever, Ashby, and custom platforms (like Apple's proprietary search API), processes the data through a local SQLite diff engine, and leverages local LLMs (via Ollama) to extract structured insights without leaking data to third-party providers.

## Supported Platforms & Limitations

> [!IMPORTANT]
> **openHunt currently supports Workday, Greenhouse, Lever, Ashby, and Apple job boards.**
> Other custom career portals (such as Intuit's Radancy/Avature setup) are not supported. If a company uses a custom domain or an unsupported ATS, the discovery tool will fail to find a supported board.

## Confirmed Target Companies

The following companies have been successfully discovered and are confirmed in the database:

| Company Name | Platform | Tenant / Identifier |
| --- | --- | --- |
| Adobe | Workday | `adobe` |
| Apple | Apple | `apple` |
| Broadcom | Workday | `broadcom` |
| Cloudera | Workday | `cloudera` |
| Coinbase | Greenhouse | `coinbase` |
| Dexcom | Workday | `dexcom` |
| Elastic | Greenhouse | `elastic` |
| Illumina | Workday | `illumina` |
| NVIDIA | Workday | `nvidia` |
| Qualcomm | Workday | `qualcomm` |
| Reddit | Greenhouse | `reddit` |
| Salesforce | Workday | `salesforce` |
| Sony PlayStation | Workday | `sonyglobal` |
| Stripe | Greenhouse | `stripe` |

## Architecture

The system operates as a multi-stage pipeline:

1.  **Concurrent Scraper**: A high-performance Go worker pool dispatches each target to its configured ATS backend and processes multiple companies simultaneously.
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

3. **Configure Ollama**:
   Copy the example environment file and set the model you have installed locally:
   ```bash
   cp .env.example .env
   ```

   Example `.env`:
   ```dotenv
   OLLAMA_API_URL="http://localhost:11434"
   OLLAMA_MODEL="gemma4:e4b"
   OPENHUNT_OUTPUT_DIR="Market-Insights"
   ```

4. **Run the engine**:
   ```bash
   go run cmd/openhunt/main.go
   ```

5. **Analyze the results**:
   Open the `Market-Insights/` folder in Obsidian to view your structured market intelligence.

### Running Tests

To run the unit tests:
```bash
go test -v ./...
```

#### Coverage Note (Go 1.25+ Toolchain Issue)
If you are running Go 1.25+ pre-release toolchains, running `go test -cover ./...` may fail with `go: no such tool "covdata"` because the compiler attempts to analyze packages without test files. To run coverage successfully, target the packages with tests explicitly:
```bash
go test -v -cover ./internal/db ./internal/discovery ./internal/scraper ./internal/tui
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

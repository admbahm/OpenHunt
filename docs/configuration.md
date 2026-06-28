# Configuration Guide

`openHunt` can be configured using a local `.env` file, environment variables, and by seeding/editing target companies in the database.

## Configuration Precedence

Configuration values are resolved in the following order (highest precedence first):

1. Environment variables exported by the shell
2. Values defined in `.env`
3. Built-in defaults

## Environment Variables

| Variable | Description | Default |
| --- | --- | --- |
| `OLLAMA_API_URL` | The endpoint URL of the Ollama server. | `http://localhost:11434` |
| `OLLAMA_MODEL` | The model name loaded in Ollama to use for processing job listings. | `llama3` |
| `OPENHUNT_OUTPUT_DIR` | Directory where generated market intelligence notes are written. This can point to an Obsidian vault or any directory on disk. | `Market-Insights` |

### Supported Models

OpenHunt works with any Ollama model capable of instruction following.

Common choices include:

- `gemma4:e4b` (recommended)
- `llama3`
- `qwen3`
- `mistral`

### Local `.env`

Create a local `.env` file from the committed example:

```bash
cp .env.example .env
```

Then edit `.env` for your local Ollama setup:

```dotenv
OLLAMA_API_URL="http://localhost:11434"
OLLAMA_MODEL="gemma4:e4b"
OPENHUNT_OUTPUT_DIR="Market-Insights"
```

The `.env` file is intentionally gitignored. If a variable is already exported in the shell that launches `openHunt`, that exported value takes precedence over `.env`.

### Example running with one-off custom settings:

```bash
OLLAMA_API_URL="http://localhost:11434" OLLAMA_MODEL="llama2:latest" go run cmd/openhunt/main.go
```

## Quick Start

```bash
cp .env.example .env

go run cmd/openhunt/main.go
```

By default OpenHunt will:

1. Connect to Ollama.
2. Read the configured target companies.
3. Harvest job postings.
4. Generate markdown market intelligence notes.

## Target Companies Configuration

Target companies are stored in the application's database. The default installation uses SQLite (`database/openhunt.db`). The table is defined as:

```sql
CREATE TABLE target_companies (
    name TEXT PRIMARY KEY,
    tenant TEXT,
    site TEXT,
    base_url TEXT,
    platform TEXT DEFAULT 'workday'
);
```

### 1. Dynamic Discovery Tool

`openHunt` includes an automated discovery CLI utility that searches the web, detects the correct ATS provider and metadata for a company, and saves it directly to your targets table.

#### Usage:
```bash
go run cmd/discover/main.go [options] "<Company Name>"
```

#### Options:
- `-y`: Auto-save the discovered target configuration without prompting for confirmation.
- `-db <path>`: Custom path to your openHunt SQLite database (defaults to `database/openhunt.db`).

#### Examples:
```bash
# Discover and prompt before saving
go run cmd/discover/main.go "Seismic"

# Discover and auto-save
go run cmd/discover/main.go -y "ClickUp"
```

---

### 2. Manual Target Administration (SQLite)

For custom target entries, you can modify the SQLite database directly from your shell.

#### View All Configured Targets:
```bash
sqlite3 database/openhunt.db "SELECT * FROM target_companies;"
```

#### Add a Greenhouse Company:
```bash
sqlite3 database/openhunt.db "INSERT INTO target_companies (name, platform, tenant) VALUES ('Stripe', 'greenhouse', 'stripe');"
```

#### Add a Workday Company:
```bash
sqlite3 database/openhunt.db "INSERT INTO target_companies (name, platform, tenant, site, base_url) VALUES ('Illumina', 'workday', 'illumina', 'illumina-careers', 'https://illumina.wd1.myworkdayjobs.com/en-US/illumina-careers/');"
```

#### Delete a Target:
```bash
sqlite3 database/openhunt.db "DELETE FROM target_companies WHERE name = 'Seismic';"
```

---

### 3. Target Validation & Health Checks

Before executing a large run, you can check that your configured targets are valid, reachable, and parsing correctly using the validation command:

```bash
# Validate all configured companies
go run cmd/validate/main.go

# Validate a specific company
go run cmd/validate/main.go --company "Apple"

# Validate by platform
go run cmd/validate/main.go --platform "workday"
```

The validation command returns a concise validation summary and supports `--json` for machine-readable output.

---

### Seeding Targets

Target companies are seeded automatically during initialization if the `target_companies` table is empty. The default seed targets can be customized in Go source code inside [internal/db/db.go](internal/db/db.go):

## Troubleshooting

### Ollama connection refused

Verify Ollama is running:

```bash
ollama serve
```

### Model not found

List installed models:

```bash
ollama list
```

Pull a model:

```bash
ollama pull gemma4:e4b
```

### Output directory not created

Verify the process has permission to write to the configured output directory.

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

Target companies are stored in the application's database. The default installation uses SQLite (database/openhunt.db). The table is defined as:

```sql
CREATE TABLE target_companies (
    name TEXT PRIMARY KEY,
    tenant TEXT,
    site TEXT,
    base_url TEXT,
    platform TEXT DEFAULT 'workday'
);
```

### Seeding Targets

Target companies are seeded automatically during initialization if the table is empty. The pre-seeded targets can be modified in [internal/db/db.go](internal/db/db.go):

- **Name**: Human-readable company name.
- **Tenant**: Workday tenant subdomain.
- **Site**: Workday site path identifier.
- **BaseURL**: Main public careers landing page URL used for harvesting CSRF cookies.
- **Platform**: Supported scraper backends:

| Platform | Description |
| --- | --- |
| `workday` | Company-hosted Workday careers sites |
| `greenhouse` | Public Greenhouse job boards |
| `lever` | Public Lever job boards |
| `ashby` | Public Ashby job boards |

For Greenhouse, Lever, and Ashby targets, `tenant` is the public board token/slug; `site` and `base_url` may be empty.

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

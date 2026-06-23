# Configuration Guide

`openHunt` can be configured using a local `.env` file, environment variables, and by seeding/editing target companies in the database.

## Environment Variables

| Variable | Description | Default |
| --- | --- | --- |
| `OLLAMA_API_URL` | The endpoint URL of the Ollama server. | `http://localhost:11434` |
| `OLLAMA_MODEL` | The model name loaded in Ollama to use for processing job listings. | `llama3` |
| `OPENHUNT_OUTPUT_DIR` | Directory where Obsidian markdown exports are written. | `Market-Insights` |

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

## Target Companies Configuration

Target companies are stored in the `target_companies` table in SQLite database (`database/openhunt.db`). The table is defined as:

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
- **Platform**: Scraper backend to use: `workday`, `greenhouse`, `lever`, or `ashby`.

For Greenhouse, Lever, and Ashby targets, `tenant` is the public board token/slug; `site` and `base_url` may be empty.

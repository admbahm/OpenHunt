# Configuration Guide

`openHunt` can be configured using environment variables and by seeding/editing target companies in the database.

## Environment Variables

| Variable | Description | Default |
| --- | --- | --- |
| `OLLAMA_API_URL` | The endpoint URL of the Ollama server. | `http://localhost:11434` |
| `OLLAMA_MODEL` | The model name loaded in Ollama to use for processing job listings. | `llama3` |

### Example running with custom settings:

```bash
OLLAMA_API_URL="http://192.168.50.88:11434" OLLAMA_MODEL="llama2:latest" go run cmd/openhunt/main.go
```

## Target Companies Configuration

Target companies are stored in the `target_companies` table in SQLite database (`database/openhunt.db`). The table is defined as:

```sql
CREATE TABLE target_companies (
    name TEXT PRIMARY KEY,
    tenant TEXT,
    site TEXT,
    base_url TEXT
);
```

### Seeding Targets

Target companies are seeded automatically during initialization if the table is empty. The pre-seeded targets can be modified in [internal/db/db.go](file:///Users/adam/dev/cross/OpenHunt/internal/db/db.go#L103-L117):

- **Name**: Human-readable company name.
- **Tenant**: Workday tenant subdomain.
- **Site**: Workday site path identifier.
- **BaseURL**: Main public careers landing page URL used for harvesting CSRF cookies.

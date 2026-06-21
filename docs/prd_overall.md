# Overall PRD: openHunt Market Intelligence Engine

## 1. Product Vision & Mission
`openHunt` is a sovereign, local-first market intelligence engine designed to counteract asymmetric automated screening pipelines. It empowers job seekers by automating the collection, processing, and analysis of job market data using a private, secure, local-first architecture. 

Rather than relying on third-party job boards that sell user data and limit access to raw information, `openHunt` lets users curate their own market intelligence directly on their local machines.

---

## 2. Core Features & Capabilities

### 2.1 Concurrent Stateful Scraping
- **Target Integrations**: Deep integration with modern Applicant Tracking Systems (ATS) starting with Workday (CXS endpoints) and Greenhouse.
- **Stateful Ingestion**: Extract cookies and CSRF tokens by performing a landing page hand-shake before requesting API endpoints to avoid bot-detection barriers.
- **Pagination support**: Loop page-by-page to guarantee that zero job listings are missed.
- **Polite Harvesting**: Enforce delays (jittered backoffs) and configurable concurrency limits to avoid target rate-limiting.

### 2.2 Local SQLite Deduplication
- **Diff Engine**: A local SQLite database acts as a delta/diff engine. It ensures that only newly listed jobs are pushed forward to the resource-intensive AI processing phase.
- **Data Persistence**: Stores job listings, company details, extraction dates, and completed AI intelligence analysis profiles.

### 2.3 Sequential AI Intelligence Pipeline
- **Local Inference**: Integrate with local LLMs (via Ollama API) to extract critical insights from raw text:
  - Base salary ranges (`base_salary_min`, `base_salary_max`).
  - Key technical stack components.
  - Regulatory or security gates (e.g., ISO, FDA, secret clearance).
  - Role classification (Individual Contributor vs. Management).
- **Fallback Resilience**: In the event that Ollama is unreachable or down, a built-in heuristics engine parses strings using regex and keyword search patterns to guarantee database and vault updates.

### 2.4 Obsidian Vault Export
- **Atomic Markdown Output**: Output active jobs under `@Active` and closed jobs under `@Closed` directories in an Obsidian-ready vault structure.
- **YAML Frontmatter**: Standardize YAML-compliant keys for robust indexing, filtering, and graph-view exploration in Obsidian.

### 2.5 Local Web Dashboard UI (Upcoming)
- **Log Monitor**: Visual output of the scraping process, active connections, and database status.
- **Jobs Explorer**: Filter, search, and sort collected postings by company, salary, role type, and tech stack.
- **Trend Charts**: Basic visualization of salary bands and requested technologies.

---

## 3. Tech Stack Requirements
- **Language**: Go 1.25+ for concurrent scheduling and ingestion performance.
- **Database**: SQLite, using a CGO-free driver (`github.com/ncruces/go-sqlite3/driver`) to simplify portability.
- **AI/LLM**: Ollama (`localhost:11434`), default model: `llama3`.
- **Knowledge Base**: Markdown flat-files, organized under standard Obsidian vaults.
- **Web Interface**: Lightweight web dashboard running on a local port.

---

## 4. Security & Sovereignty Rules
- **No Third-Party Leaks**: All raw text processing and analysis must happen on the user's localhost. Job data must never be sent to third-party LLM providers.
- **Local SQLite Storage**: The SQLite database remains strictly local.

---
name: openhunt-dev
description: Guidelines and scripts for developing, testing, and debugging the openHunt job market intelligence engine.
---

# openHunt Developer Skill

Use this skill when developing, testing, or debugging components of openHunt (the scraper, database layer, AI pipeline, or manual ingestion tools).

## System Architecture Reference

- **Database**: SQLite (`database/openhunt.db`).
  - Table `jobs`: primary storage for scraped and analyzed jobs.
  - Table `target_companies`: contains seeded target configurations.
- **Obsidian Vault**: Markdown files are written to `Market-Insights/@Active` and `@Closed` paths.

## Formatting Guidelines for Obsidian Exports

All markdown files exported to the Obsidian vault must include YAML frontmatter in this format:
```yaml
---
job_id: <requisition_id>
company: <company_name>
title: "<job_title>"
location: "<location>"
posted_at: <posted_time_string>
salary_min: <integer>
salary_max: <integer>
role_type: "<Individual Contributor | Management>"
tech_stack: [<comma_separated_strings>]
regulatory_gates: [<comma_separated_strings>]
scraped_at: <timestamp>
---
```

## Adding Target Companies
To add a target company, update the seed list in `internal/db/db.go` and ensure it gets written during initialization, or insert it directly via SQL query into `database/openhunt.db`.
Target platforms are:
- `workday`
- `greenhouse`

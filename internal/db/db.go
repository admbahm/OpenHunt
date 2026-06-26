package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openhunt/openhunt/internal/scraper"
	"github.com/openhunt/openhunt/internal/telemetry"

	_ "github.com/ncruces/go-sqlite3/driver"
)

// Store defines the interface for database operations.
type Store interface {
	Close() error
	IsJobNew(jobID string) (bool, error)
	SaveJob(company string, job scraper.JobListing, analysis *telemetry.AnalysisResult) error
}

// SQLStore implements the Store interface using SQLite.
type SQLStore struct {
	db *sql.DB
}

// NewSQLStore initializes a new SQLite database and applies migrations.
func NewSQLStore(dbPath string) (*SQLStore, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open the database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLStore{db: db}

	// Run migrations
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return store, nil
}

// Close closes the database connection.
func (s *SQLStore) Close() error {
	return s.db.Close()
}

// migrate handles schema initialization and updates.
func (s *SQLStore) migrate() error {
	// Initial schema
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		title TEXT,
		company TEXT,
		location TEXT,
		url TEXT,
		description TEXT,
		posted_at TEXT,
		scraped_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		salary_min INTEGER,
		salary_max INTEGER,
		tech_stack TEXT,
		regulatory_gates TEXT,
		role_type TEXT
	);

	CREATE TABLE IF NOT EXISTS target_companies (
		name TEXT PRIMARY KEY,
		tenant TEXT,
		site TEXT,
		base_url TEXT,
		platform TEXT DEFAULT 'workday'
	);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// CREATE TABLE IF NOT EXISTS does not update databases created by older
	// versions. Apply additive migrations explicitly so existing user data can
	// be opened by newer builds without requiring the database to be deleted.
	if err := s.addColumnIfMissing("jobs", "description", "TEXT"); err != nil {
		return err
	}

	return nil
}

func (s *SQLStore) addColumnIfMissing(table, column, definition string) error {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("failed to inspect %s schema: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal any
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); err != nil {
			return fmt.Errorf("failed to inspect %s columns: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to inspect %s columns: %w", table, err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("failed to close %s schema inspection: %w", table, err)
	}

	if _, err := s.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)); err != nil {
		return fmt.Errorf("failed to add %s.%s: %w", table, column, err)
	}
	return nil
}

// SeedTargets populates the target_companies table with initial data if empty.
func (s *SQLStore) SeedTargets() error {
	targets := []scraper.TargetCompany{
		{
			Name:     "Illumina",
			Tenant:   "illumina",
			Site:     "illumina-careers",
			BaseURL:  "https://illumina.wd1.myworkdayjobs.com/en-US/illumina-careers/",
			Platform: "workday",
		},
		{
			Name:     "Dexcom",
			Tenant:   "dexcom",
			Site:     "Dexcom",
			BaseURL:  "https://dexcom.wd1.myworkdayjobs.com/Dexcom/",
			Platform: "workday",
		},
		{
			Name:     "Stripe",
			Tenant:   "stripe",
			Platform: "greenhouse",
		},
		{
			Name:     "Apple",
			Tenant:   "apple",
			BaseURL:  "https://jobs.apple.com",
			Platform: "apple",
		},
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO target_companies (name, tenant, site, base_url, platform) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range targets {
		if _, err := stmt.Exec(t.Name, t.Tenant, t.Site, t.BaseURL, t.Platform); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetTargets retrieves all target companies from the database.
func (s *SQLStore) GetTargets() ([]scraper.TargetCompany, error) {
	rows, err := s.db.Query("SELECT name, tenant, site, base_url, platform FROM target_companies")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []scraper.TargetCompany
	for rows.Next() {
		var t scraper.TargetCompany
		if err := rows.Scan(&t.Name, &t.Tenant, &t.Site, &t.BaseURL, &t.Platform); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// AddTarget inserts a new target company or updates it if it exists.
func (s *SQLStore) AddTarget(t scraper.TargetCompany) error {
	query := `
	INSERT INTO target_companies (name, tenant, site, base_url, platform)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(name) DO UPDATE SET
		tenant = excluded.tenant,
		site = excluded.site,
		base_url = excluded.base_url,
		platform = excluded.platform
	`
	_, err := s.db.Exec(query, t.Name, t.Tenant, t.Site, t.BaseURL, t.Platform)
	return err
}

// IsJobNew checks if a job ID already exists in the database.
func (s *SQLStore) IsJobNew(jobID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM jobs WHERE id = ?)`
	err := s.db.QueryRow(query, jobID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

// SaveJob inserts a new job and its analysis into the database.
func (s *SQLStore) SaveJob(company string, job scraper.JobListing, analysis *telemetry.AnalysisResult) error {
	query := `
	INSERT INTO jobs (id, title, company, location, url, description, posted_at, salary_min, salary_max, tech_stack, regulatory_gates, role_type)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	// Simple comma-separated strings for slices for now
	var techStack, regGates string
	if analysis != nil {
		techStack = join(analysis.TechStack, ", ")
		regGates = join(analysis.RegulatoryGates, ", ")
	} else {
		// Provide default empty analysis if none exists to avoid nil dereference
		analysis = &telemetry.AnalysisResult{
			RoleType: "Unknown",
		}
	}

	_, err := s.db.Exec(query,
		job.JobID,
		job.Title,
		company,
		job.LocationsText,
		job.ExternalPath,
		job.Description,
		job.PostedOn,
		analysis.BaseSalaryMin,
		analysis.BaseSalaryMax,
		techStack,
		regGates,
		analysis.RoleType,
	)
	return err
}

func join(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	res := s[0]
	for i := 1; i < len(s); i++ {
		res += sep + s[i]
	}
	return res
}

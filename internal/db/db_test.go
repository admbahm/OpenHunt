package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/openhunt/openhunt/internal/scraper"
)

func TestSQLStore(t *testing.T) {
	// Create a temporary database file
	tempDir, err := os.MkdirTemp("", "openhunt-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	store, err := NewSQLStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLStore: %v", err)
	}
	defer store.Close()

	// Test SeedTargets and GetTargets
	err = store.SeedTargets()
	if err != nil {
		t.Fatalf("SeedTargets failed: %v", err)
	}

	targets, err := store.GetTargets()
	if err != nil {
		t.Fatalf("GetTargets failed: %v", err)
	}

	if len(targets) == 0 {
		t.Error("Expected at least one target after seeding, got 0")
	}

	// Test IsJobNew and SaveJob
	jobID := "test-job-1"
	isNew, err := store.IsJobNew(jobID)
	if err != nil {
		t.Fatalf("IsJobNew failed: %v", err)
	}
	if !isNew {
		t.Error("Expected job to be new")
	}

	job := scraper.JobListing{
		JobID: jobID,
		Title: "Test Title",
	}
	err = store.SaveJob("Test Company", job, nil)
	if err != nil {
		t.Fatalf("SaveJob failed: %v", err)
	}

	isNew, err = store.IsJobNew(jobID)
	if err != nil {
		t.Fatalf("IsJobNew failed: %v", err)
	}
	if isNew {
		t.Error("Expected job NOT to be new after saving")
	}
}

func TestNewSQLStoreMigratesLegacyJobsDescriptionColumn(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	legacyDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	_, err = legacyDB.Exec(`
		CREATE TABLE jobs (
			id TEXT PRIMARY KEY,
			title TEXT,
			company TEXT,
			location TEXT,
			url TEXT,
			posted_at TEXT,
			scraped_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			salary_min INTEGER,
			salary_max INTEGER,
			tech_stack TEXT,
			regulatory_gates TEXT,
			role_type TEXT
		);
	`)
	if err != nil {
		legacyDB.Close()
		t.Fatalf("create legacy schema: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	store, err := NewSQLStore(dbPath)
	if err != nil {
		t.Fatalf("migrate legacy database: %v", err)
	}
	defer store.Close()

	job := scraper.JobListing{
		JobID:       "legacy-migration-job",
		Title:       "Engineer",
		Description: "Migrated description",
	}
	if err := store.SaveJob("Acme", job, nil); err != nil {
		t.Fatalf("save job after migration: %v", err)
	}

	var description string
	if err := store.db.QueryRow(
		"SELECT description FROM jobs WHERE id = ?",
		job.JobID,
	).Scan(&description); err != nil {
		t.Fatalf("read migrated description: %v", err)
	}
	if description != job.Description {
		t.Fatalf("description = %q, want %q", description, job.Description)
	}
}

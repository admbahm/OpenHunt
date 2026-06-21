package db

import (
	"os"
	"path/filepath"
	"testing"

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

package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// Store defines the interface for database operations.
type Store interface {
	Close() error
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
		posted_at DATETIME,
		scraped_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}
	return nil
}

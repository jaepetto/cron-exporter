package model

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type Database struct {
	db *sqlx.DB
}

// NewDatabase creates a new Database instance
func NewDatabase(dbPath string) (*Database, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sqlx.Open("sqlite", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{db: db}

	// Run migrations
	if err := database.RunMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	logrus.WithField("db_path", dbPath).Info("database initialized successfully")
	return database, nil
}

// GetDB returns the underlying sqlx database connection
func (d *Database) GetDB() *sqlx.DB {
	return d.db
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// RunMigrations applies all pending migrations
func (d *Database) RunMigrations() error {
	// Create migrations table if it doesn't exist
	if err := d.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	appliedMigrations, err := d.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Get all migration files
	migrationFiles, err := d.getMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to get migration files: %w", err)
	}

	// Apply pending migrations
	for _, filename := range migrationFiles {
		if _, applied := appliedMigrations[filename]; !applied {
			if err := d.applyMigration(filename); err != nil {
				return fmt.Errorf("failed to apply migration %s: %w", filename, err)
			}
		}
	}
	return nil
}

// createMigrationsTable creates the migrations tracking table
func (d *Database) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			filename TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err := d.db.Exec(query)
	return err
}

// getAppliedMigrations returns a map of applied migration filenames
func (d *Database) getAppliedMigrations() (map[string]bool, error) {
	query := `SELECT filename FROM migrations`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, err
		}
		applied[filename] = true
	}

	return applied, rows.Err()
}

// getMigrationFiles returns sorted list of migration files
func (d *Database) getMigrationFiles() ([]string, error) {
	// For embedded migrations, we'll define them inline
	// In a real application, you might read from a migrations/ directory
	migrations := []string{
		"001_create_jobs_table.sql",
		"002_create_job_results_table.sql",
		"003_add_api_key_to_jobs.sql",
		"004_add_job_id_column.sql",
	}

	sort.Strings(migrations)
	return migrations, nil
}

// applyMigration applies a single migration
func (d *Database) applyMigration(filename string) error {
	sql, err := d.getMigrationSQL(filename)
	if err != nil {
		return fmt.Errorf("failed to get migration SQL: %w", err)
	}

	// Execute the migration in a transaction
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(sql); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration as applied
	if _, err := tx.Exec("INSERT INTO migrations (filename) VALUES (?)", filename); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	logrus.WithField("migration", filename).Info("migration applied successfully")
	return nil
}

// getMigrationSQL returns the SQL for a migration file
func (d *Database) getMigrationSQL(filename string) (string, error) {
	switch filename {
	case "001_create_jobs_table.sql":
		return `
			CREATE TABLE jobs (
				name TEXT NOT NULL,
				host TEXT NOT NULL,
				automatic_failure_threshold INTEGER NOT NULL DEFAULT 3600,
				labels TEXT NOT NULL DEFAULT '{}',
				status TEXT NOT NULL DEFAULT 'active',
				last_reported_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (name, host)
			);

			CREATE INDEX idx_jobs_status ON jobs(status);
			CREATE INDEX idx_jobs_last_reported ON jobs(last_reported_at);
		`, nil

	case "002_create_job_results_table.sql":
		return `
			CREATE TABLE job_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				job_name TEXT NOT NULL,
				host TEXT NOT NULL,
				status TEXT NOT NULL,
				labels TEXT DEFAULT '{}',
				duration INTEGER,
				output TEXT,
				timestamp DATETIME NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (job_name, host) REFERENCES jobs(name, host) ON DELETE CASCADE
			);

			CREATE INDEX idx_job_results_job ON job_results(job_name, host);
			CREATE INDEX idx_job_results_timestamp ON job_results(timestamp);
			CREATE INDEX idx_job_results_status ON job_results(status);
		`, nil

	case "003_add_api_key_to_jobs.sql":
		return `
			ALTER TABLE jobs ADD COLUMN api_key TEXT;
			CREATE UNIQUE INDEX idx_jobs_api_key ON jobs(api_key) WHERE api_key IS NOT NULL;
		`, nil

	case "004_add_job_id_column.sql":
		return `
			-- Migration: Add ID column to jobs table and update primary key
			-- This migration adds an auto-incrementing ID column and changes the primary key
			-- from (name, host) composite key to just ID for better referencing

			-- Create new table with ID as primary key
			CREATE TABLE jobs_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				host TEXT NOT NULL,
				api_key TEXT,
				automatic_failure_threshold INTEGER NOT NULL DEFAULT 3600,
				labels TEXT NOT NULL DEFAULT '{}',
				status TEXT NOT NULL DEFAULT 'active',
				last_reported_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(name, host) -- Keep name+host combination unique
			);

			-- Copy data from old table to new table (if it exists)
			INSERT INTO jobs_new (name, host, api_key, automatic_failure_threshold, labels, status, last_reported_at, created_at, updated_at)
			SELECT name, host, api_key, automatic_failure_threshold, labels, status, last_reported_at, created_at, updated_at
			FROM jobs
			WHERE EXISTS (SELECT 1 FROM sqlite_master WHERE type='table' AND name='jobs');

			-- Drop old table
			DROP TABLE IF EXISTS jobs;

			-- Rename new table
			ALTER TABLE jobs_new RENAME TO jobs;

			-- Create indexes
			CREATE INDEX idx_jobs_status ON jobs(status);
			CREATE INDEX idx_jobs_last_reported ON jobs(last_reported_at);
			CREATE INDEX idx_jobs_name_host ON jobs(name, host);

			-- Update job_results table to reference job by ID instead of name+host
			-- First, add job_id column to job_results table
			ALTER TABLE job_results ADD COLUMN job_id INTEGER REFERENCES jobs(id);

			-- Create index on job_id for better performance
			CREATE INDEX idx_job_results_job_id ON job_results(job_id);
		`, nil

	default:
		return "", fmt.Errorf("unknown migration file: %s", filename)
	}
}

// JobResultStore provides database operations for job results
type JobResultStore struct {
	db *sqlx.DB
}

// NewJobResultStore creates a new JobResultStore instance
func NewJobResultStore(db *sqlx.DB) *JobResultStore {
	return &JobResultStore{db: db}
}

// CreateJobResult creates a new job result record
func (s *JobResultStore) CreateJobResult(result *JobResult) error {
	labelsJSON := "{}"
	if result.Labels != nil {
		if bytes, err := json.Marshal(result.Labels); err == nil {
			labelsJSON = string(bytes)
		}
	}

	query := `
		INSERT INTO job_results (job_name, host, status, labels, duration, output, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, result.JobName, result.Host, result.Status, labelsJSON, result.Duration, result.Output, result.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to create job result: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"job_name": result.JobName,
		"host":     result.Host,
		"status":   result.Status,
		"duration": result.Duration,
	}).Info("job result recorded")

	return nil
}

// GetJobResults retrieves job results with optional filtering
func (s *JobResultStore) GetJobResults(jobName, host string, limit int) ([]*JobResult, error) {
	query := `
		SELECT job_name, host, status, labels, duration, output, timestamp
		FROM job_results
		WHERE job_name = ? AND host = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := s.db.Queryx(query, jobName, host, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get job results: %w", err)
	}
	defer rows.Close()

	var results []*JobResult
	for rows.Next() {
		result := &JobResult{}
		var labelsJSON string
		var output sql.NullString
		var duration sql.NullInt64

		err := rows.Scan(&result.JobName, &result.Host, &result.Status, &labelsJSON, &duration, &output, &result.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job result row: %w", err)
		}

		if duration.Valid {
			result.Duration = int(duration.Int64)
		}
		if output.Valid {
			result.Output = output.String
		}

		if labelsJSON != "{}" && labelsJSON != "" {
			if err := json.Unmarshal([]byte(labelsJSON), &result.Labels); err != nil {
				logrus.WithError(err).Warn("failed to unmarshal job result labels")
			}
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

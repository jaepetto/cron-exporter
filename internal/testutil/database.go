package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaepetto/cron-exporter/pkg/model"
	"github.com/stretchr/testify/require"
)

// TestDatabase provides utilities for creating and managing test databases
type TestDatabase struct {
	DB   *model.Database
	Path string
	t    *testing.T
}

// NewTestDatabase creates a new temporary SQLite database for testing
func NewTestDatabase(t *testing.T) *TestDatabase {
	// Create temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize database
	db, err := model.NewDatabase(dbPath)
	require.NoError(t, err, "Failed to create test database")

	return &TestDatabase{
		DB:   db,
		Path: dbPath,
		t:    t,
	}
}

// NewInMemoryTestDatabase creates an in-memory SQLite database for testing
func NewInMemoryTestDatabase(t *testing.T) *TestDatabase {
	// Use in-memory database
	db, err := model.NewDatabase(":memory:")
	require.NoError(t, err, "Failed to create in-memory test database")

	return &TestDatabase{
		DB:   db,
		Path: ":memory:",
		t:    t,
	}
}

// Close closes the test database and cleans up resources
func (td *TestDatabase) Close() {
	if td.DB != nil {
		if err := td.DB.Close(); err != nil {
			// In test context, just ignore database close errors
			_ = err
		}
	}

	// Clean up file if it's not in-memory
	if td.Path != ":memory:" && td.Path != "" {
		if err := os.Remove(td.Path); err != nil {
			// In test context, just ignore file removal errors
			_ = err
		}
	}
}

// GetJobStore returns a JobStore instance for the test database
func (td *TestDatabase) GetJobStore() *model.JobStore {
	return model.NewJobStore(td.DB.GetDB())
}

// GetJobResultStore returns a JobResultStore instance for the test database
func (td *TestDatabase) GetJobResultStore() *model.JobResultStore {
	return model.NewJobResultStore(td.DB.GetDB())
}

// Exec executes a SQL statement on the test database
func (td *TestDatabase) Exec(query string, args ...interface{}) {
	_, err := td.DB.GetDB().Exec(query, args...)
	require.NoError(td.t, err, fmt.Sprintf("Failed to execute query: %s", query))
}

// Query executes a query and returns the result set
func (td *TestDatabase) Query(query string, args ...interface{}) *sql.Rows {
	rows, err := td.DB.GetDB().Query(query, args...)
	require.NoError(td.t, err, fmt.Sprintf("Failed to execute query: %s", query))
	return rows
}

// QueryRow executes a query and returns a single row
func (td *TestDatabase) QueryRow(query string, args ...interface{}) *sql.Row {
	return td.DB.GetDB().QueryRow(query, args...)
}

// SeedTestData adds some basic test data to the database
func (td *TestDatabase) SeedTestData() {
	jobStore := td.GetJobStore()

	// Create test jobs
	testJobs := []struct {
		name      string
		host      string
		threshold int
		labels    map[string]string
		status    string
		apiKey    string
	}{
		{
			name:      "backup",
			host:      "db1",
			threshold: 3600,
			labels:    map[string]string{"env": "prod", "type": "backup"},
			status:    "active",
			apiKey:    "cm_test_backup_key",
		},
		{
			name:      "log-rotation",
			host:      "web1",
			threshold: 1800,
			labels:    map[string]string{"env": "prod", "type": "maintenance"},
			status:    "active",
			apiKey:    "cm_test_logrotation_key",
		},
		{
			name:      "maintenance-job",
			host:      "app1",
			threshold: 7200,
			labels:    map[string]string{"env": "staging"},
			status:    "maintenance",
			apiKey:    "cm_test_maintenance_key",
		},
	}

	for _, job := range testJobs {
		err := jobStore.CreateJob(&model.Job{
			Name:                      job.name,
			Host:                      job.host,
			AutomaticFailureThreshold: job.threshold,
			Labels:                    job.labels,
			Status:                    job.status,
			ApiKey:                    job.apiKey,
		})
		require.NoError(td.t, err, fmt.Sprintf("Failed to create test job: %s@%s", job.name, job.host))
	}
}

// CountJobs returns the number of jobs in the database
func (td *TestDatabase) CountJobs() int {
	var count int
	err := td.DB.GetDB().QueryRow("SELECT COUNT(*) FROM jobs").Scan(&count)
	require.NoError(td.t, err, "Failed to count jobs")
	return count
}

// CountJobResults returns the number of job results in the database
func (td *TestDatabase) CountJobResults() int {
	var count int
	err := td.DB.GetDB().QueryRow("SELECT COUNT(*) FROM job_results").Scan(&count)
	require.NoError(td.t, err, "Failed to count job results")
	return count
}

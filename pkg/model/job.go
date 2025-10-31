package model

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// Job represents a cron job definition with its configuration and status
type Job struct {
	ID                        int               `json:"id" db:"id"` // Auto-incrementing primary key
	Name                      string            `json:"job_name" db:"name"`
	Host                      string            `json:"host" db:"host"`
	ApiKey                    string            `json:"api_key,omitempty" db:"api_key"`                               // Per-job API key for authentication
	AutomaticFailureThreshold int               `json:"automatic_failure_threshold" db:"automatic_failure_threshold"` // Seconds since last result
	Labels                    map[string]string `json:"labels" db:"labels"`                                           // Arbitrary user labels
	Status                    string            `json:"status" db:"status"`                                           // "active", "maintenance", "paused"
	LastReportedAt            time.Time         `json:"last_reported_at" db:"last_reported_at"`                       // For auto-failure logic
	CreatedAt                 time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time         `json:"updated_at" db:"updated_at"`
}

// JobResult represents a job execution result submission
type JobResult struct {
	JobName   string            `json:"job_name"`
	Host      string            `json:"host"`
	Status    string            `json:"status"` // "success", "failure"
	Labels    map[string]string `json:"labels,omitempty"`
	Duration  int               `json:"duration,omitempty"` // Execution duration in seconds
	Output    string            `json:"output,omitempty"`   // Optional execution output
	Timestamp time.Time         `json:"timestamp"`
}

// JobStore provides database operations for jobs
type JobStore struct {
	db *sql.DB
}

// NewJobStore creates a new JobStore instance
func NewJobStore(db *sql.DB) *JobStore {
	return &JobStore{db: db}
}

// CreateJob creates a new job in the database
func (s *JobStore) CreateJob(job *Job) error {
	labelsJSON, err := json.Marshal(job.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now

	query := `
		INSERT INTO jobs (name, host, api_key, automatic_failure_threshold, labels, status, last_reported_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(query, job.Name, job.Host, job.ApiKey, job.AutomaticFailureThreshold, string(labelsJSON), job.Status, job.LastReportedAt, job.CreatedAt, job.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	// Get the auto-generated ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get job ID: %w", err)
	}
	job.ID = int(id)

	logrus.WithFields(logrus.Fields{
		"job_name": job.Name,
		"host":     job.Host,
		"status":   job.Status,
	}).Info("job created successfully")

	return nil
}

// GetJobByID retrieves a job by its ID
func (s *JobStore) GetJobByID(id int) (*Job, error) {
	query := `
		SELECT id, name, host, api_key, automatic_failure_threshold, labels, status, last_reported_at, created_at, updated_at
		FROM jobs
		WHERE id = ?
	`

	row := s.db.QueryRow(query, id)

	job := &Job{}
	var labelsJSON string
	var apiKeyNull sql.NullString

	err := row.Scan(&job.ID, &job.Name, &job.Host, &apiKeyNull, &job.AutomaticFailureThreshold, &labelsJSON, &job.Status, &job.LastReportedAt, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found with ID: %d", id)
		}
		return nil, fmt.Errorf("failed to get job by ID: %w", err)
	}

	if apiKeyNull.Valid {
		job.ApiKey = apiKeyNull.String
	}

	if err := json.Unmarshal([]byte(labelsJSON), &job.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}

	return job, nil
}

// GetJob retrieves a job by name and host (kept for backward compatibility)
func (s *JobStore) GetJob(name, host string) (*Job, error) {
	query := `
		SELECT id, name, host, api_key, automatic_failure_threshold, labels, status, last_reported_at, created_at, updated_at
		FROM jobs
		WHERE name = ? AND host = ?
	`

	row := s.db.QueryRow(query, name, host)

	job := &Job{}
	var labelsJSON string
	var apiKeyNull sql.NullString

	err := row.Scan(&job.ID, &job.Name, &job.Host, &apiKeyNull, &job.AutomaticFailureThreshold, &labelsJSON, &job.Status, &job.LastReportedAt, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s@%s", name, host)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if apiKeyNull.Valid {
		job.ApiKey = apiKeyNull.String
	}

	if err := json.Unmarshal([]byte(labelsJSON), &job.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}

	return job, nil
}

// ListJobs retrieves all jobs with optional label filtering
func (s *JobStore) ListJobs(labelFilters map[string]string) ([]*Job, error) {
	query := `
		SELECT id, name, host, api_key, automatic_failure_threshold, labels, status, last_reported_at, created_at, updated_at
		FROM jobs
		ORDER BY id
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job := &Job{}
		var labelsJSON string
		var apiKeyNull sql.NullString

		err := rows.Scan(&job.ID, &job.Name, &job.Host, &apiKeyNull, &job.AutomaticFailureThreshold, &labelsJSON, &job.Status, &job.LastReportedAt, &job.CreatedAt, &job.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job row: %w", err)
		}

		if apiKeyNull.Valid {
			job.ApiKey = apiKeyNull.String
		}

		if err := json.Unmarshal([]byte(labelsJSON), &job.Labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}

		// Apply label filters if provided
		if len(labelFilters) > 0 {
			match := true
			for key, value := range labelFilters {
				if job.Labels[key] != value {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job rows: %w", err)
	}

	return jobs, nil
}

// UpdateJobByID updates an existing job by ID
func (s *JobStore) UpdateJobByID(job *Job) error {
	labelsJSON, err := json.Marshal(job.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	job.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE jobs
		SET name = ?, host = ?, api_key = ?, automatic_failure_threshold = ?, labels = ?, status = ?, last_reported_at = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := s.db.Exec(query, job.Name, job.Host, job.ApiKey, job.AutomaticFailureThreshold, string(labelsJSON), job.Status, job.LastReportedAt, job.UpdatedAt, job.ID)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job not found with ID: %d", job.ID)
	}

	logrus.WithFields(logrus.Fields{
		"job_id":   job.ID,
		"job_name": job.Name,
		"host":     job.Host,
		"status":   job.Status,
	}).Info("job updated successfully")

	return nil
}

// UpdateJob updates an existing job (kept for backward compatibility)
func (s *JobStore) UpdateJob(job *Job) error {
	labelsJSON, err := json.Marshal(job.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	job.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE jobs
		SET api_key = ?, automatic_failure_threshold = ?, labels = ?, status = ?, last_reported_at = ?, updated_at = ?
		WHERE name = ? AND host = ?
	`

	result, err := s.db.Exec(query, job.ApiKey, job.AutomaticFailureThreshold, string(labelsJSON), job.Status, job.LastReportedAt, job.UpdatedAt, job.Name, job.Host)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job not found: %s@%s", job.Name, job.Host)
	}

	logrus.WithFields(logrus.Fields{
		"job_name": job.Name,
		"host":     job.Host,
		"status":   job.Status,
	}).Info("job updated successfully")

	return nil
}

// DeleteJobByID removes a job from the database by ID
func (s *JobStore) DeleteJobByID(id int) error {
	query := `DELETE FROM jobs WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job not found with ID: %d", id)
	}

	logrus.WithFields(logrus.Fields{
		"job_id": id,
	}).Info("job deleted successfully")

	return nil
}

// DeleteJob removes a job from the database (kept for backward compatibility)
func (s *JobStore) DeleteJob(name, host string) error {
	query := `DELETE FROM jobs WHERE name = ? AND host = ?`

	result, err := s.db.Exec(query, name, host)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job not found: %s@%s", name, host)
	}

	logrus.WithFields(logrus.Fields{
		"job_name": name,
		"host":     host,
	}).Info("job deleted successfully")

	return nil
}

// UpdateJobLastReported updates the last_reported_at timestamp for a job
func (s *JobStore) UpdateJobLastReported(name, host string, timestamp time.Time) error {
	query := `
		UPDATE jobs
		SET last_reported_at = ?, updated_at = ?
		WHERE name = ? AND host = ?
	`

	now := time.Now().UTC()
	result, err := s.db.Exec(query, timestamp, now, name, host)
	if err != nil {
		return fmt.Errorf("failed to update job last reported: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job not found: %s@%s", name, host)
	}

	return nil
}

// GetJobByApiKey retrieves a job by its API key
func (s *JobStore) GetJobByApiKey(apiKey string) (*Job, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}

	query := `
		SELECT id, name, host, api_key, automatic_failure_threshold, labels, status, last_reported_at, created_at, updated_at
		FROM jobs
		WHERE api_key = ?
	`

	row := s.db.QueryRow(query, apiKey)

	job := &Job{}
	var labelsJSON string
	var apiKeyNull sql.NullString

	err := row.Scan(&job.ID, &job.Name, &job.Host, &apiKeyNull, &job.AutomaticFailureThreshold, &labelsJSON, &job.Status, &job.LastReportedAt, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found for API key")
		}
		return nil, fmt.Errorf("failed to get job by API key: %w", err)
	}

	if apiKeyNull.Valid {
		job.ApiKey = apiKeyNull.String
	}

	if err := json.Unmarshal([]byte(labelsJSON), &job.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}

	return job, nil
}

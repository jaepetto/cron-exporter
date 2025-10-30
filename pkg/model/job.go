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
	Name                      string            `json:"job_name" db:"name"`
	Host                      string            `json:"host" db:"host"`
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
		INSERT INTO jobs (name, host, automatic_failure_threshold, labels, status, last_reported_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query, job.Name, job.Host, job.AutomaticFailureThreshold, string(labelsJSON), job.Status, job.LastReportedAt, job.CreatedAt, job.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"job_name": job.Name,
		"host":     job.Host,
		"status":   job.Status,
	}).Info("job created successfully")

	return nil
}

// GetJob retrieves a job by name and host
func (s *JobStore) GetJob(name, host string) (*Job, error) {
	query := `
		SELECT name, host, automatic_failure_threshold, labels, status, last_reported_at, created_at, updated_at
		FROM jobs
		WHERE name = ? AND host = ?
	`

	row := s.db.QueryRow(query, name, host)

	job := &Job{}
	var labelsJSON string

	err := row.Scan(&job.Name, &job.Host, &job.AutomaticFailureThreshold, &labelsJSON, &job.Status, &job.LastReportedAt, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s@%s", name, host)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if err := json.Unmarshal([]byte(labelsJSON), &job.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}

	return job, nil
}

// ListJobs retrieves all jobs with optional label filtering
func (s *JobStore) ListJobs(labelFilters map[string]string) ([]*Job, error) {
	query := `
		SELECT name, host, automatic_failure_threshold, labels, status, last_reported_at, created_at, updated_at
		FROM jobs
		ORDER BY name, host
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

		err := rows.Scan(&job.Name, &job.Host, &job.AutomaticFailureThreshold, &labelsJSON, &job.Status, &job.LastReportedAt, &job.CreatedAt, &job.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job row: %w", err)
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

// UpdateJob updates an existing job
func (s *JobStore) UpdateJob(job *Job) error {
	labelsJSON, err := json.Marshal(job.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	job.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE jobs
		SET automatic_failure_threshold = ?, labels = ?, status = ?, last_reported_at = ?, updated_at = ?
		WHERE name = ? AND host = ?
	`

	result, err := s.db.Exec(query, job.AutomaticFailureThreshold, string(labelsJSON), job.Status, job.LastReportedAt, job.UpdatedAt, job.Name, job.Host)
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

// DeleteJob removes a job from the database
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

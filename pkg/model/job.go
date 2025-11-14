package model

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
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

// JobSearchCriteria represents advanced search and filtering options for jobs
type JobSearchCriteria struct {
	// Text search fields
	Query string `json:"query,omitempty"` // Search across name, host, and labels

	// Specific field filters
	Name   string `json:"name,omitempty"`   // Filter by job name (partial match)
	Host   string `json:"host,omitempty"`   // Filter by host (partial match)
	Status string `json:"status,omitempty"` // Filter by job status (exact match)

	// Label filters
	Labels map[string]string `json:"labels,omitempty"` // Filter by labels (exact match)

	// Time-based filters
	LastReportedBefore *time.Time `json:"last_reported_before,omitempty"` // Jobs reported before this time
	LastReportedAfter  *time.Time `json:"last_reported_after,omitempty"`  // Jobs reported after this time

	// Pagination
	Page     int `json:"page,omitempty"`      // Page number (1-based)
	PageSize int `json:"page_size,omitempty"` // Number of items per page
}

// JobSearchResult represents paginated search results
type JobSearchResult struct {
	Jobs        []*Job `json:"jobs"`
	TotalCount  int    `json:"total_count"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
	TotalPages  int    `json:"total_pages"`
	HasNext     bool   `json:"has_next"`
	HasPrevious bool   `json:"has_previous"`
	SearchQuery string `json:"search_query,omitempty"`
}

// JobStore provides database operations for jobs
type JobStore struct {
	db *sqlx.DB
}

// NewJobStore creates a new JobStore instance
func NewJobStore(db *sqlx.DB) *JobStore {
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

	job := &Job{}
	var labelsJSON string
	var apiKeyNull sql.NullString

	err := s.db.QueryRowx(query, id).Scan(&job.ID, &job.Name, &job.Host, &apiKeyNull, &job.AutomaticFailureThreshold, &labelsJSON, &job.Status, &job.LastReportedAt, &job.CreatedAt, &job.UpdatedAt)
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

	job := &Job{}
	var labelsJSON string
	var apiKeyNull sql.NullString

	err := s.db.QueryRowx(query, name, host).Scan(&job.ID, &job.Name, &job.Host, &apiKeyNull, &job.AutomaticFailureThreshold, &labelsJSON, &job.Status, &job.LastReportedAt, &job.CreatedAt, &job.UpdatedAt)
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

	rows, err := s.db.Queryx(query)
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

// SearchJobs performs advanced search with filtering and pagination
func (s *JobStore) SearchJobs(criteria *JobSearchCriteria) (*JobSearchResult, error) {
	if criteria == nil {
		criteria = &JobSearchCriteria{}
	}

	// Set default pagination values
	if criteria.Page <= 0 {
		criteria.Page = 1
	}
	if criteria.PageSize <= 0 {
		criteria.PageSize = 25 // Default page size
	}

	// Build the WHERE clause dynamically
	var whereConditions []string
	var args []interface{}
	argIndex := 0

	// Handle text query search across name, host, and labels
	if criteria.Query != "" {
		// Search in name, host, and labels JSON
		whereConditions = append(whereConditions,
			"(name LIKE ? OR host LIKE ? OR labels LIKE ?)")
		searchTerm := "%" + criteria.Query + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
		argIndex += 3
	}

	// Handle specific field filters
	if criteria.Name != "" {
		whereConditions = append(whereConditions, "name LIKE ?")
		args = append(args, "%"+criteria.Name+"%")
		argIndex++
	}

	if criteria.Host != "" {
		whereConditions = append(whereConditions, "host LIKE ?")
		args = append(args, "%"+criteria.Host+"%")
		argIndex++
	}

	if criteria.Status != "" {
		whereConditions = append(whereConditions, "status = ?")
		args = append(args, criteria.Status)
		argIndex++
	}

	// Handle time-based filters
	if criteria.LastReportedBefore != nil {
		whereConditions = append(whereConditions, "last_reported_at < ?")
		args = append(args, criteria.LastReportedBefore.UTC())
		argIndex++
	}

	if criteria.LastReportedAfter != nil {
		whereConditions = append(whereConditions, "last_reported_at > ?")
		args = append(args, criteria.LastReportedAfter.UTC())
		argIndex++
	}

	// Build the complete WHERE clause
	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// First, get the total count for pagination
	countQuery := "SELECT COUNT(*) FROM jobs " + whereClause

	var totalCount int
	err := s.db.Get(&totalCount, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to count jobs: %w", err)
	}

	// Calculate pagination values
	totalPages := (totalCount + criteria.PageSize - 1) / criteria.PageSize
	offset := (criteria.Page - 1) * criteria.PageSize

	// Build the main query with pagination
	query := "SELECT id, name, host, api_key, automatic_failure_threshold, labels, status, last_reported_at, created_at, updated_at FROM jobs " + whereClause + " ORDER BY id LIMIT ? OFFSET ?"

	// Add pagination parameters
	paginationArgs := append(args, criteria.PageSize, offset)

	rows, err := s.db.Queryx(query, paginationArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to search jobs: %w", err)
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

		// Apply label filters if provided (post-query filtering for complex JSON matching)
		if len(criteria.Labels) > 0 {
			match := true
			for key, value := range criteria.Labels {
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

	// Build the result
	result := &JobSearchResult{
		Jobs:        jobs,
		TotalCount:  totalCount,
		Page:        criteria.Page,
		PageSize:    criteria.PageSize,
		TotalPages:  totalPages,
		HasNext:     criteria.Page < totalPages,
		HasPrevious: criteria.Page > 1,
		SearchQuery: criteria.Query,
	}

	return result, nil
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

	job := &Job{}
	var labelsJSON string
	var apiKeyNull sql.NullString

	err := s.db.QueryRowx(query, apiKey).Scan(&job.ID, &job.Name, &job.Host, &apiKeyNull, &job.AutomaticFailureThreshold, &labelsJSON, &job.Status, &job.LastReportedAt, &job.CreatedAt, &job.UpdatedAt)
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

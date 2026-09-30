// Package jobservice provides application-level job validation and operations.
package jobservice

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jaepetto/cron-exporter/pkg/model"
	"github.com/jaepetto/cron-exporter/pkg/util"
)

const (
	defaultFailureThreshold = 3600
	defaultPageSize         = 25
	maximumPageSize         = 100
)

// ErrorKind identifies an application error category suitable for HTTP mapping.
type ErrorKind string

const (
	ErrorValidation ErrorKind = "validation"
	ErrorNotFound   ErrorKind = "not_found"
	ErrorConflict   ErrorKind = "conflict"
	ErrorInternal   ErrorKind = "internal"
)

// Error is a stable application error with optional field-level validation details.
type Error struct {
	Kind    ErrorKind         `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
	Cause   error             `json:"-"`
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// CreateJobInput contains values accepted when creating a job.
type CreateJobInput struct {
	Name                      string            `json:"job_name"`
	Host                      string            `json:"host"`
	APIKey                    string            `json:"api_key,omitempty"`
	AutomaticFailureThreshold int               `json:"automatic_failure_threshold,omitempty"`
	Labels                    map[string]string `json:"labels,omitempty"`
	Status                    string            `json:"status,omitempty"`
}

// UpdateJobInput contains optional values accepted when updating a job.
type UpdateJobInput struct {
	Name                      *string            `json:"job_name,omitempty"`
	Host                      *string            `json:"host,omitempty"`
	APIKey                    *string            `json:"api_key,omitempty"`
	AutomaticFailureThreshold *int               `json:"automatic_failure_threshold,omitempty"`
	Labels                    *map[string]string `json:"labels,omitempty"`
	Status                    *string            `json:"status,omitempty"`
}

// Service applies job business rules before persistence.
type Service struct {
	store *model.JobStore
}

// New creates a job application service backed by store.
func New(store *model.JobStore) *Service {
	return &Service{store: store}
}

// Create validates and persists a job.
func (s *Service) Create(input CreateJobInput) (*model.Job, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Host = strings.TrimSpace(input.Host)
	if input.AutomaticFailureThreshold == 0 {
		input.AutomaticFailureThreshold = defaultFailureThreshold
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Labels == nil {
		input.Labels = map[string]string{}
	}

	fields := validateJob(input.Name, input.Host, input.AutomaticFailureThreshold, input.Status)
	if len(fields) > 0 {
		return nil, validationError(fields)
	}

	apiKey := input.APIKey
	if apiKey == "" {
		generated, err := util.GenerateAPIKey()
		if err != nil {
			return nil, internalError("failed to generate job API key", err)
		}
		apiKey = generated
	}

	job := &model.Job{
		Name:                      input.Name,
		Host:                      input.Host,
		ApiKey:                    apiKey,
		AutomaticFailureThreshold: input.AutomaticFailureThreshold,
		Labels:                    input.Labels,
		Status:                    input.Status,
		LastReportedAt:            time.Now().UTC(),
	}
	if err := s.store.CreateJob(job); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, &Error{Kind: ErrorConflict, Message: "job already exists", Cause: err}
		}
		return nil, internalError("failed to create job", err)
	}

	return job, nil
}

// Get returns a job by ID.
func (s *Service) Get(id int) (*model.Job, error) {
	if id <= 0 {
		return nil, validationError(map[string]string{"id": "must be a positive integer"})
	}

	job, err := s.store.GetJobByID(id)
	if err != nil {
		return nil, classifyStoreError("failed to get job", err)
	}
	return job, nil
}

// List searches jobs using bounded pagination.
func (s *Service) List(criteria model.JobSearchCriteria) (*model.JobSearchResult, error) {
	if criteria.Page <= 0 {
		criteria.Page = 1
	}
	if criteria.PageSize <= 0 {
		criteria.PageSize = defaultPageSize
	}
	if criteria.PageSize > maximumPageSize {
		return nil, validationError(map[string]string{"page_size": fmt.Sprintf("must be at most %d", maximumPageSize)})
	}
	if criteria.Status != "" && !isValidStatus(criteria.Status) {
		return nil, validationError(map[string]string{"status": "must be active, maintenance, or paused"})
	}

	result, err := s.store.SearchJobs(&criteria)
	if err != nil {
		return nil, internalError("failed to list jobs", err)
	}
	return result, nil
}

// Update validates and persists a partial job update.
func (s *Service) Update(id int, input UpdateJobInput) (*model.Job, error) {
	job, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		job.Name = strings.TrimSpace(*input.Name)
	}
	if input.Host != nil {
		job.Host = strings.TrimSpace(*input.Host)
	}
	if input.APIKey != nil && *input.APIKey != "" {
		job.ApiKey = *input.APIKey
	}
	if input.AutomaticFailureThreshold != nil {
		job.AutomaticFailureThreshold = *input.AutomaticFailureThreshold
	}
	if input.Labels != nil {
		job.Labels = *input.Labels
	}
	if input.Status != nil {
		job.Status = *input.Status
	}

	fields := validateJob(job.Name, job.Host, job.AutomaticFailureThreshold, job.Status)
	if len(fields) > 0 {
		return nil, validationError(fields)
	}
	if job.Labels == nil {
		job.Labels = map[string]string{}
	}

	if err := s.store.UpdateJobByID(job); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, &Error{Kind: ErrorConflict, Message: "job already exists", Cause: err}
		}
		return nil, classifyStoreError("failed to update job", err)
	}
	return job, nil
}

// Delete removes a job by ID and returns its prior value for event publication.
func (s *Service) Delete(id int) (*model.Job, error) {
	job, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := s.store.DeleteJobByID(id); err != nil {
		return nil, classifyStoreError("failed to delete job", err)
	}
	return job, nil
}

// ToggleMaintenance switches a job between active and maintenance status.
func (s *Service) ToggleMaintenance(id int) (*model.Job, error) {
	job, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	status := "maintenance"
	if job.Status == "maintenance" {
		status = "active"
	}
	return s.Update(id, UpdateJobInput{Status: &status})
}

// IsKind reports whether err is a job service error of kind.
func IsKind(err error, kind ErrorKind) bool {
	var serviceError *Error
	return errors.As(err, &serviceError) && serviceError.Kind == kind
}

func validateJob(name, host string, threshold int, status string) map[string]string {
	fields := map[string]string{}
	if name == "" {
		fields["job_name"] = "is required"
	}
	if host == "" {
		fields["host"] = "is required"
	}
	if threshold <= 0 {
		fields["automatic_failure_threshold"] = "must be greater than zero"
	}
	if !isValidStatus(status) {
		fields["status"] = "must be active, maintenance, or paused"
	}
	return fields
}

func isValidStatus(status string) bool {
	return status == "active" || status == "maintenance" || status == "paused"
}

func validationError(fields map[string]string) *Error {
	return &Error{Kind: ErrorValidation, Message: "job validation failed", Fields: fields}
}

func internalError(message string, cause error) *Error {
	return &Error{Kind: ErrorInternal, Message: message, Cause: cause}
}

func classifyStoreError(message string, err error) *Error {
	if strings.Contains(err.Error(), "not found") {
		return &Error{Kind: ErrorNotFound, Message: "job not found", Cause: err}
	}
	return internalError(message, err)
}

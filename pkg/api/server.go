package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jaep/cron-exporter/pkg/config"
	"github.com/jaep/cron-exporter/pkg/metrics"
	"github.com/jaep/cron-exporter/pkg/model"
	"github.com/sirupsen/logrus"
)

// Server represents the HTTP API server
type Server struct {
	config         *config.Config
	jobStore       *model.JobStore
	jobResultStore *model.JobResultStore
	metrics        *metrics.Collector
}

// NewServer creates a new API server instance
func NewServer(cfg *config.Config, jobStore *model.JobStore, jobResultStore *model.JobResultStore, metricsCollector *metrics.Collector) *Server {
	return &Server{
		config:         cfg,
		jobStore:       jobStore,
		jobResultStore: jobResultStore,
		metrics:        metricsCollector,
	}
}

// Handler returns the HTTP handler for the server
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/job", s.withAuth(s.handleJob))
	mux.HandleFunc("/api/job/", s.withAuth(s.handleJobByID))
	mux.HandleFunc("/api/job-result", s.withAuth(s.handleJobResult))

	// Metrics endpoint
	mux.HandleFunc(s.config.Metrics.Path, s.handleMetrics)

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// Add request logging middleware
	return s.withLogging(mux)
}

// withAuth provides authentication middleware
func (s *Server) withAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth in development mode
		if s.config.Database.Path == "/tmp/cronmetrics_dev.db" {
			handler(w, r)
			return
		}

		// Check for API key in Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			s.writeErrorResponse(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		// Extract Bearer token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			s.writeErrorResponse(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		// Check if token is valid
		isAdmin := s.isValidAdminAPIKey(token)
		isRegular := s.isValidAPIKey(token)

		if !isAdmin && !isRegular {
			s.writeErrorResponse(w, http.StatusUnauthorized, "invalid API key")
			return
		}

		// Add auth info to request context for admin-only operations
		if isAdmin {
			r.Header.Set("X-Auth-Level", "admin")
		} else {
			r.Header.Set("X-Auth-Level", "regular")
		}

		handler(w, r)
	}
}

// withLogging provides request logging middleware
func (s *Server) withLogging(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a wrapped response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		handler.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		logrus.WithFields(logrus.Fields{
			"method":         r.Method,
			"path":           r.URL.Path,
			"status":         wrapped.statusCode,
			"duration_ms":    duration.Milliseconds(),
			"remote_addr":    r.RemoteAddr,
			"user_agent":     r.UserAgent(),
			"content_length": r.ContentLength,
		}).Info("http request")
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// handleJob handles job CRUD operations
func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateJob(w, r)
	case http.MethodGet:
		s.handleListJobs(w, r)
	default:
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleJobByID handles operations on specific jobs
func (s *Server) handleJobByID(w http.ResponseWriter, r *http.Request) {
	// Extract job name and host from path
	path := strings.TrimPrefix(r.URL.Path, "/api/job/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid job path format (expected /api/job/{name}/{host})")
		return
	}

	jobName := parts[0]
	jobHost := parts[1]

	switch r.Method {
	case http.MethodGet:
		s.handleGetJob(w, r, jobName, jobHost)
	case http.MethodPut:
		s.handleUpdateJob(w, r, jobName, jobHost)
	case http.MethodDelete:
		s.handleDeleteJob(w, r, jobName, jobHost)
	default:
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleCreateJob creates a new job
func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	// Only admin can create jobs
	if r.Header.Get("X-Auth-Level") != "admin" {
		s.writeErrorResponse(w, http.StatusForbidden, "admin access required")
		return
	}

	var job model.Job
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	// Validate required fields
	if job.Name == "" || job.Host == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "job name and host are required")
		return
	}

	// Set defaults
	if job.AutomaticFailureThreshold == 0 {
		job.AutomaticFailureThreshold = 3600
	}
	if job.Status == "" {
		job.Status = "active"
	}
	if job.Labels == nil {
		job.Labels = make(map[string]string)
	}
	job.LastReportedAt = time.Now().UTC()

	if err := s.jobStore.CreateJob(&job); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			s.writeErrorResponse(w, http.StatusConflict, "job already exists")
			return
		}
		s.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to create job: %v", err))
		return
	}

	s.writeJSONResponse(w, http.StatusCreated, job)
}

// handleListJobs lists all jobs with optional filtering
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	// Parse label filters from query parameters
	labelFilters := make(map[string]string)
	for key, values := range r.URL.Query() {
		if strings.HasPrefix(key, "label.") {
			labelKey := strings.TrimPrefix(key, "label.")
			if len(values) > 0 {
				labelFilters[labelKey] = values[0]
			}
		}
	}

	jobs, err := s.jobStore.ListJobs(labelFilters)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to list jobs: %v", err))
		return
	}

	s.writeJSONResponse(w, http.StatusOK, jobs)
}

// handleGetJob gets a specific job
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request, jobName, jobHost string) {
	job, err := s.jobStore.GetJob(jobName, jobHost)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeErrorResponse(w, http.StatusNotFound, "job not found")
			return
		}
		s.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get job: %v", err))
		return
	}

	s.writeJSONResponse(w, http.StatusOK, job)
}

// handleUpdateJob updates a job
func (s *Server) handleUpdateJob(w http.ResponseWriter, r *http.Request, jobName, jobHost string) {
	// Only admin can update jobs
	if r.Header.Get("X-Auth-Level") != "admin" {
		s.writeErrorResponse(w, http.StatusForbidden, "admin access required")
		return
	}

	// Get existing job
	existingJob, err := s.jobStore.GetJob(jobName, jobHost)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeErrorResponse(w, http.StatusNotFound, "job not found")
			return
		}
		s.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to get job: %v", err))
		return
	}

	var updateData model.Job
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	// Update only provided fields
	if updateData.AutomaticFailureThreshold > 0 {
		existingJob.AutomaticFailureThreshold = updateData.AutomaticFailureThreshold
	}
	if updateData.Labels != nil {
		existingJob.Labels = updateData.Labels
	}
	if updateData.Status != "" {
		existingJob.Status = updateData.Status
	}

	if err := s.jobStore.UpdateJob(existingJob); err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to update job: %v", err))
		return
	}

	s.writeJSONResponse(w, http.StatusOK, existingJob)
}

// handleDeleteJob deletes a job
func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request, jobName, jobHost string) {
	// Only admin can delete jobs
	if r.Header.Get("X-Auth-Level") != "admin" {
		s.writeErrorResponse(w, http.StatusForbidden, "admin access required")
		return
	}

	if err := s.jobStore.DeleteJob(jobName, jobHost); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeErrorResponse(w, http.StatusNotFound, "job not found")
			return
		}
		s.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete job: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleJobResult handles job result submissions
func (s *Server) handleJobResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var result model.JobResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	// Validate required fields
	if result.JobName == "" || result.Host == "" || result.Status == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "job_name, host, and status are required")
		return
	}

	// Validate status
	if result.Status != "success" && result.Status != "failure" {
		s.writeErrorResponse(w, http.StatusBadRequest, "status must be 'success' or 'failure'")
		return
	}

	// Set timestamp if not provided
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now().UTC()
	}

	// Store the job result
	if err := s.jobResultStore.CreateJobResult(&result); err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to store job result: %v", err))
		return
	}

	// Update job's last reported timestamp
	if err := s.jobStore.UpdateJobLastReported(result.JobName, result.Host, result.Timestamp); err != nil {
		// Log error but don't fail the request
		logrus.WithError(err).WithFields(logrus.Fields{
			"job_name": result.JobName,
			"host":     result.Host,
		}).Warn("failed to update job last reported timestamp")
	}

	s.writeJSONResponse(w, http.StatusCreated, map[string]string{
		"status": "recorded",
		"job":    fmt.Sprintf("%s@%s", result.JobName, result.Host),
	})
}

// handleMetrics serves Prometheus metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	metrics, err := s.metrics.Gather()
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to gather metrics: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(metrics))
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "0.1.0",
	}

	s.writeJSONResponse(w, http.StatusOK, health)
}

// isValidAPIKey checks if the provided token is a valid API key
func (s *Server) isValidAPIKey(token string) bool {
	for _, key := range s.config.Security.APIKeys {
		if key == token {
			return true
		}
	}
	return false
}

// isValidAdminAPIKey checks if the provided token is a valid admin API key
func (s *Server) isValidAdminAPIKey(token string) bool {
	for _, key := range s.config.Security.AdminAPIKeys {
		if key == token {
			return true
		}
	}
	return false
}

// writeJSONResponse writes a JSON response
func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logrus.WithError(err).Error("failed to encode JSON response")
	}
}

// writeErrorResponse writes an error response
func (s *Server) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	errorResponse := map[string]string{
		"error":     message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	s.writeJSONResponse(w, statusCode, errorResponse)
}

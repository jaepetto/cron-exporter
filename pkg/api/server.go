package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/jaepetto/cron-exporter/pkg/config"
	"github.com/jaepetto/cron-exporter/pkg/metrics"
	"github.com/jaepetto/cron-exporter/pkg/model"
	"github.com/jaepetto/cron-exporter/pkg/util"
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
	mux.HandleFunc("/api/job-result", s.withJobAuth(s.handleJobResult))

	// Metrics endpoint
	mux.HandleFunc(s.config.Metrics.Path, s.handleMetrics)

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// Swagger UI and OpenAPI spec
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/api/openapi.yaml"), // The URL pointing to the OpenAPI spec
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("swagger-ui"),
	))
	mux.HandleFunc("/api/openapi.yaml", s.handleOpenAPISpec)

	// Add request logging middleware
	return s.withLogging(mux)
}

// withAuth provides authentication middleware for admin operations
func (s *Server) withAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth in development mode
		if s.config.Database.Path == "/tmp/cronmetrics_dev.db" {
			handler(w, r)
			return
		}

		// Get API key from header
		apiKey := s.extractAPIKey(r)
		if apiKey == "" {
			s.writeErrorResponse(w, http.StatusUnauthorized, "missing or invalid API key")
			return
		}

		// Check if token is valid admin key
		if !s.isValidAdminAPIKey(apiKey) {
			s.writeErrorResponse(w, http.StatusUnauthorized, "admin access required")
			return
		}

		// Add auth info to request context
		r.Header.Set("X-Auth-Level", "admin")
		handler(w, r)
	}
}

// withJobAuth provides authentication middleware for job result submissions
func (s *Server) withJobAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth in development mode
		if s.config.Database.Path == "/tmp/cronmetrics_dev.db" {
			handler(w, r)
			return
		}

		// Get API key from header
		apiKey := s.extractAPIKey(r)
		if apiKey == "" {
			s.writeErrorResponse(w, http.StatusUnauthorized, "missing or invalid API key")
			return
		}

		// Validate API key by looking up the associated job
		job, err := s.jobStore.GetJobByApiKey(apiKey)
		if err != nil {
			s.writeErrorResponse(w, http.StatusUnauthorized, "invalid API key")
			return
		}

		// Add job info to request context for validation
		r.Header.Set("X-Auth-Job-Name", job.Name)
		r.Header.Set("X-Auth-Job-Host", job.Host)
		r.Header.Set("X-Auth-Level", "job")

		handler(w, r)
	}
}

// extractAPIKey extracts API key from various header formats
func (s *Server) extractAPIKey(r *http.Request) string {
	// Try X-API-Key header first (preferred for job submissions)
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	// Fall back to Authorization Bearer format (for admin operations)
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Extract Bearer token
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return ""
	}

	return token
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

// handleJobByID handles operations on specific jobs using job ID
func (s *Server) handleJobByID(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/job/")

	if path == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid job path format (expected /api/job/{id})")
		return
	}

	// Parse job ID
	jobID := 0
	if _, err := fmt.Sscanf(path, "%d", &jobID); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid job ID format (must be a number)")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetJobByID(w, r, jobID)
	case http.MethodPut:
		s.handleUpdateJobByID(w, r, jobID)
	case http.MethodDelete:
		s.handleDeleteJobByID(w, r, jobID)
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

	// Generate API key if not provided
	if job.ApiKey == "" {
		apiKey, err := util.GenerateAPIKey()
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate API key: %v", err))
			return
		}
		job.ApiKey = apiKey
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

// handleGetJobByID retrieves a specific job by ID
func (s *Server) handleGetJobByID(w http.ResponseWriter, r *http.Request, jobID int) {
	job, err := s.jobStore.GetJobByID(jobID)
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

// handleGetJob retrieves a specific job (kept for backward compatibility)
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

// handleUpdateJobByID updates a job by ID
func (s *Server) handleUpdateJobByID(w http.ResponseWriter, r *http.Request, jobID int) {
	// Only admin can update jobs
	if r.Header.Get("X-Auth-Level") != "admin" {
		s.writeErrorResponse(w, http.StatusForbidden, "admin access required")
		return
	}

	// Get existing job
	existingJob, err := s.jobStore.GetJobByID(jobID)
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
	if updateData.Name != "" {
		existingJob.Name = updateData.Name
	}
	if updateData.Host != "" {
		existingJob.Host = updateData.Host
	}
	if updateData.ApiKey != "" {
		existingJob.ApiKey = updateData.ApiKey
	}
	if updateData.AutomaticFailureThreshold > 0 {
		existingJob.AutomaticFailureThreshold = updateData.AutomaticFailureThreshold
	}
	if updateData.Labels != nil {
		existingJob.Labels = updateData.Labels
	}
	if updateData.Status != "" {
		existingJob.Status = updateData.Status
	}

	if err := s.jobStore.UpdateJobByID(existingJob); err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to update job: %v", err))
		return
	}

	s.writeJSONResponse(w, http.StatusOK, existingJob)
}

// handleUpdateJob updates a job (kept for backward compatibility)
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
	if updateData.ApiKey != "" {
		existingJob.ApiKey = updateData.ApiKey
	}
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

// handleDeleteJobByID deletes a job by ID
func (s *Server) handleDeleteJobByID(w http.ResponseWriter, r *http.Request, jobID int) {
	// Only admin can delete jobs
	if r.Header.Get("X-Auth-Level") != "admin" {
		s.writeErrorResponse(w, http.StatusForbidden, "admin access required")
		return
	}

	if err := s.jobStore.DeleteJobByID(jobID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeErrorResponse(w, http.StatusNotFound, "job not found")
			return
		}
		s.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete job: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteJob deletes a job (kept for backward compatibility)
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

	// In non-dev mode, validate that the job result matches the authenticated job
	if s.config.Database.Path != "/tmp/cronmetrics_dev.db" {
		authJobName := r.Header.Get("X-Auth-Job-Name")
		authJobHost := r.Header.Get("X-Auth-Job-Host")

		if result.JobName != authJobName || result.Host != authJobHost {
			s.writeErrorResponse(w, http.StatusForbidden, "job result does not match authenticated job")
			return
		}
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

// handleOpenAPISpec serves the OpenAPI specification file
func (s *Server) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Find the OpenAPI spec file relative to the binary
	specPath := "docs/openapi.yaml"

	// Try to find the spec file in multiple locations
	possiblePaths := []string{
		specPath,                            // Relative to current working directory
		filepath.Join(".", specPath),        // Explicit relative path
		filepath.Join("..", specPath),       // One level up (in case running from bin/)
		filepath.Join("..", "..", specPath), // Two levels up
	}

	var content []byte
	var err error

	for _, path := range possiblePaths {
		content, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		logrus.WithError(err).Errorf("Failed to read OpenAPI spec from any of these paths: %v", possiblePaths)
		s.writeErrorResponse(w, http.StatusInternalServerError, "OpenAPI specification not found")
		return
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.WriteHeader(http.StatusOK)
	w.Write(content)
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

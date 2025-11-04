package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaepetto/cron-exporter/pkg/config"
	"github.com/jaepetto/cron-exporter/pkg/model"
	"github.com/sirupsen/logrus"
)

// Handler contains all HTTP handlers for the dashboard
type Handler struct {
	config       *config.DashboardConfig
	jobStore     *model.JobStore
	assetHandler *AssetHandler
	broadcaster  *Broadcaster
	logger       *logrus.Logger
}

// NewHandler creates a new dashboard handler
func NewHandler(config *config.DashboardConfig, jobStore *model.JobStore, logger *logrus.Logger) *Handler {
	broadcaster := NewBroadcaster(config, jobStore, logger)

	return &Handler{
		config:       config,
		jobStore:     jobStore,
		assetHandler: NewAssetHandler(),
		broadcaster:  broadcaster,
		logger:       logger,
	}
}

// ServeAssets serves embedded static assets
func (h *Handler) ServeAssets(c *gin.Context) {
	// Get the filepath parameter from Gin route
	filepath := c.Param("filepath")

	// Create a new request with the correct path
	r := c.Request.Clone(c.Request.Context())
	r.URL.Path = filepath

	h.assetHandler.ServeHTTP(c.Writer, r)
}

// JobsList displays the main jobs list page
func (h *Handler) JobsList(c *gin.Context) {
	// Use the search system with empty criteria to get all jobs with pagination
	criteria := &model.JobSearchCriteria{
		Page:     1,
		PageSize: 25, // Default page size
	}

	result, err := h.jobStore.SearchJobs(criteria)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list jobs")
		c.String(http.StatusInternalServerError, "Failed to load jobs")
		return
	}

	data := gin.H{
		"Title":        h.config.Title,
		"Jobs":         result.Jobs,
		"SearchResult": result,
		"Config":       h.config,
		"SearchQuery":  "",
		"Criteria":     criteria,
	}

	c.HTML(http.StatusOK, "jobs.html", data)
}

// JobCreateForm displays the job creation form
func (h *Handler) JobCreateForm(c *gin.Context) {
	data := gin.H{
		"Title":  h.config.Title,
		"Config": h.config,
	}

	c.HTML(http.StatusOK, "job_form.html", data)
}

// JobCreate handles creating a new job
func (h *Handler) JobCreate(c *gin.Context) {
	job := &model.Job{
		Name:                      c.PostForm("name"),
		Host:                      c.PostForm("host"),
		Status:                    c.PostForm("status"),
		AutomaticFailureThreshold: 3600, // Default
	}

	// Parse automatic failure threshold
	if thresholdStr := c.PostForm("automatic_failure_threshold"); thresholdStr != "" {
		if threshold, err := strconv.Atoi(thresholdStr); err == nil && threshold > 0 {
			job.AutomaticFailureThreshold = threshold
		}
	}

	// Parse labels JSON
	if labelsStr := c.PostForm("labels"); labelsStr != "" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(labelsStr), &labels); err == nil {
			job.Labels = labels
		}
	}

	// Validate required fields
	if job.Name == "" || job.Host == "" {
		c.String(http.StatusBadRequest, "Name and host are required")
		return
	}

	// Create job
	if err := h.jobStore.CreateJob(job); err != nil {
		h.logger.WithError(err).Error("Failed to create job")
		c.String(http.StatusInternalServerError, "Failed to create job")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"job_name": job.Name,
		"host":     job.Host,
	}).Info("Job created via dashboard")

	// Broadcast job created event
	h.broadcaster.BroadcastJobCreated(job)

	// Redirect to job detail page
	c.Redirect(http.StatusFound, h.config.Path+"/jobs/"+strconv.Itoa(job.ID))
}

// JobDetail displays job details
func (h *Handler) JobDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid job ID")
		return
	}

	job, err := h.jobStore.GetJobByID(id)
	if err != nil {
		h.logger.WithError(err).WithField("job_id", id).Error("Failed to get job")
		c.String(http.StatusNotFound, "Job not found")
		return
	}

	data := gin.H{
		"Title":  h.config.Title,
		"Job":    job,
		"Config": h.config,
	}

	c.HTML(http.StatusOK, "job_detail.html", data)
}

// JobEditForm displays the job edit form
func (h *Handler) JobEditForm(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid job ID")
		return
	}

	job, err := h.jobStore.GetJobByID(id)
	if err != nil {
		h.logger.WithError(err).WithField("job_id", id).Error("Failed to get job")
		c.String(http.StatusNotFound, "Job not found")
		return
	}

	data := gin.H{
		"Title":  h.config.Title,
		"Job":    job,
		"Config": h.config,
		"Edit":   true,
	}

	c.HTML(http.StatusOK, "job_form.html", data)
}

// JobUpdate handles updating a job
func (h *Handler) JobUpdate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid job ID")
		return
	}

	// Get existing job
	job, err := h.jobStore.GetJobByID(id)
	if err != nil {
		h.logger.WithError(err).WithField("job_id", id).Error("Failed to get job for update")
		c.String(http.StatusNotFound, "Job not found")
		return
	}

	// Update fields from form
	if name := c.PostForm("name"); name != "" {
		job.Name = name
	}
	if host := c.PostForm("host"); host != "" {
		job.Host = host
	}
	if status := c.PostForm("status"); status != "" {
		job.Status = status
	}

	// Parse automatic failure threshold
	if thresholdStr := c.PostForm("automatic_failure_threshold"); thresholdStr != "" {
		if threshold, err := strconv.Atoi(thresholdStr); err == nil && threshold > 0 {
			job.AutomaticFailureThreshold = threshold
		}
	}

	// Parse labels JSON
	if labelsStr := c.PostForm("labels"); labelsStr != "" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(labelsStr), &labels); err == nil {
			job.Labels = labels
		}
	}

	// Update job
	if err := h.jobStore.UpdateJob(job); err != nil {
		h.logger.WithError(err).WithField("job_id", id).Error("Failed to update job")
		c.String(http.StatusInternalServerError, "Failed to update job")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"job_id":   job.ID,
		"job_name": job.Name,
		"host":     job.Host,
	}).Info("Job updated via dashboard")

	// Broadcast job updated event
	h.broadcaster.BroadcastJobUpdated(job)

	// Redirect to job detail page
	c.Redirect(http.StatusFound, h.config.Path+"/jobs/"+strconv.Itoa(job.ID))
}

// JobDelete handles deleting a job
func (h *Handler) JobDelete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid job ID")
		return
	}

	// Get job for logging
	job, err := h.jobStore.GetJobByID(id)
	if err != nil {
		h.logger.WithError(err).WithField("job_id", id).Error("Failed to get job for deletion")
		c.String(http.StatusNotFound, "Job not found")
		return
	}

	// Delete job
	if err := h.jobStore.DeleteJob(job.Name, job.Host); err != nil {
		h.logger.WithError(err).WithField("job_id", id).Error("Failed to delete job")
		c.String(http.StatusInternalServerError, "Failed to delete job")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"job_id":   job.ID,
		"job_name": job.Name,
		"host":     job.Host,
	}).Info("Job deleted via dashboard")

	// Broadcast job deleted event
	h.broadcaster.BroadcastJobDeleted(job.ID, job.Name, job.Host)

	// Redirect to jobs list
	c.Redirect(http.StatusFound, h.config.Path+"/jobs")
}

// JobsListAPI returns jobs list as JSON for HTMX
func (h *Handler) JobsListAPI(c *gin.Context) {
	jobs, err := h.jobStore.ListJobs(nil) // No label filters for now
	if err != nil {
		h.logger.WithError(err).Error("Failed to list jobs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

// JobStatusAPI returns job status for HTMX updates
func (h *Handler) JobStatusAPI(c *gin.Context) {
	// TODO: Implement job status API
	c.String(http.StatusNotImplemented, "Job status API not implemented yet")
}

// JobToggle handles toggling job maintenance mode
func (h *Handler) JobToggle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid job ID")
		return
	}

	// Get job
	job, err := h.jobStore.GetJobByID(id)
	if err != nil {
		h.logger.WithError(err).WithField("job_id", id).Error("Failed to get job for toggle")
		c.String(http.StatusNotFound, "Job not found")
		return
	}

	// Toggle maintenance mode
	if job.Status == "maintenance" {
		job.Status = "active"
	} else {
		job.Status = "maintenance"
	}

	// Update job
	if err := h.jobStore.UpdateJob(job); err != nil {
		h.logger.WithError(err).WithField("job_id", id).Error("Failed to toggle job status")
		c.String(http.StatusInternalServerError, "Failed to toggle job status")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"job_id":     job.ID,
		"job_name":   job.Name,
		"host":       job.Host,
		"new_status": job.Status,
	}).Info("Job status toggled via dashboard")

	// Broadcast job status change
	isFailure := false
	if job.AutomaticFailureThreshold > 0 {
		timeSinceLastReport := time.Since(job.LastReportedAt)
		if timeSinceLastReport > time.Duration(job.AutomaticFailureThreshold)*time.Second {
			isFailure = true
		}
	}
	h.broadcaster.BroadcastJobStatusChange(job, isFailure)

	// Return to job detail page
	c.Redirect(http.StatusFound, h.config.Path+"/jobs/"+strconv.Itoa(job.ID))
}

// JobSearch handles advanced job search requests with HTMX support
func (h *Handler) JobSearch(c *gin.Context) {
	// Parse search criteria from query parameters
	criteria := &model.JobSearchCriteria{
		Query:  c.Query("q"),
		Name:   c.Query("name"),
		Host:   c.Query("host"),
		Status: c.Query("status"),
	}

	// Parse pagination parameters
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			criteria.Page = page
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 && pageSize <= 100 {
			criteria.PageSize = pageSize
		}
	}

	// Parse time-based filters
	if beforeStr := c.Query("before"); beforeStr != "" {
		if before, err := time.Parse(time.RFC3339, beforeStr); err == nil {
			criteria.LastReportedBefore = &before
		}
	}
	if afterStr := c.Query("after"); afterStr != "" {
		if after, err := time.Parse(time.RFC3339, afterStr); err == nil {
			criteria.LastReportedAfter = &after
		}
	}

	// Parse label filters (JSON format: {"key1":"value1","key2":"value2"})
	if labelsStr := c.Query("labels"); labelsStr != "" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(labelsStr), &labels); err == nil {
			criteria.Labels = labels
		}
	}

	// Perform the search
	result, err := h.jobStore.SearchJobs(criteria)
	if err != nil {
		h.logger.WithError(err).Error("Failed to search jobs")
		c.String(http.StatusInternalServerError, "Failed to search jobs")
		return
	}

	// Check if this is an HTMX request for partial updates
	if c.GetHeader("HX-Request") == "true" {
		// Return just the job list table body for HTMX partial updates
		c.HTML(http.StatusOK, "job_list_partial.html", gin.H{
			"Jobs":         result.Jobs,
			"SearchResult": result,
			"Config":       h.config,
			"SearchQuery":  criteria.Query,
		})
		return
	}

	// For full page requests, return the complete jobs page with search results
	data := gin.H{
		"Title":        h.config.Title,
		"Jobs":         result.Jobs,
		"SearchResult": result,
		"Config":       h.config,
		"SearchQuery":  criteria.Query,
		"Criteria":     criteria,
	}

	c.HTML(http.StatusOK, "jobs.html", data)
}

// JobSearchAPI handles job search API requests for HTMX
func (h *Handler) JobSearchAPI(c *gin.Context) {
	// Parse search criteria from query parameters
	criteria := &model.JobSearchCriteria{
		Query:  c.Query("q"),
		Name:   c.Query("name"),
		Host:   c.Query("host"),
		Status: c.Query("status"),
	}

	// Parse pagination parameters
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			criteria.Page = page
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 && pageSize <= 100 {
			criteria.PageSize = pageSize
		}
	}

	// Parse time-based filters
	if beforeStr := c.Query("before"); beforeStr != "" {
		if before, err := time.Parse(time.RFC3339, beforeStr); err == nil {
			criteria.LastReportedBefore = &before
		}
	}
	if afterStr := c.Query("after"); afterStr != "" {
		if after, err := time.Parse(time.RFC3339, afterStr); err == nil {
			criteria.LastReportedAfter = &after
		}
	}

	// Parse label filters
	if labelsStr := c.Query("labels"); labelsStr != "" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(labelsStr), &labels); err == nil {
			criteria.Labels = labels
		}
	}

	// Perform the search
	result, err := h.jobStore.SearchJobs(criteria)
	if err != nil {
		h.logger.WithError(err).Error("Failed to search jobs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search jobs"})
		return
	}

	// Check if this is a request for HTML partial update (HTMX)
	if c.GetHeader("HX-Request") == "true" {
		// Return HTML partial for table body update
		c.HTML(http.StatusOK, "job_list_partial.html", gin.H{
			"Jobs":         result.Jobs,
			"SearchResult": result,
			"Config":       h.config,
			"SearchQuery":  criteria.Query,
			"Criteria":     criteria,
		})
		return
	}

	// Return JSON for API clients
	c.JSON(http.StatusOK, result)
}

// JobSearchWithPagination handles job search with pagination UI updates
func (h *Handler) JobSearchWithPagination(c *gin.Context) {
	// Parse search criteria from query parameters (same as JobSearchAPI)
	criteria := &model.JobSearchCriteria{
		Query:  c.Query("q"),
		Name:   c.Query("name"),
		Host:   c.Query("host"),
		Status: c.Query("status"),
	}

	// Parse pagination parameters
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			criteria.Page = page
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 && pageSize <= 100 {
			criteria.PageSize = pageSize
		}
	}

	// Parse time-based filters
	if beforeStr := c.Query("before"); beforeStr != "" {
		if before, err := time.Parse(time.RFC3339, beforeStr); err == nil {
			criteria.LastReportedBefore = &before
		}
	}
	if afterStr := c.Query("after"); afterStr != "" {
		if after, err := time.Parse(time.RFC3339, afterStr); err == nil {
			criteria.LastReportedAfter = &after
		}
	}

	// Parse label filters
	if labelsStr := c.Query("labels"); labelsStr != "" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(labelsStr), &labels); err == nil {
			criteria.Labels = labels
		}
	}

	// Perform the search
	result, err := h.jobStore.SearchJobs(criteria)
	if err != nil {
		h.logger.WithError(err).Error("Failed to search jobs")
		c.String(http.StatusInternalServerError, "Failed to search jobs")
		return
	}

	// Return both table body and pagination for HTMX multi-target updates
	data := gin.H{
		"Jobs":         result.Jobs,
		"SearchResult": result,
		"Config":       h.config,
		"SearchQuery":  criteria.Query,
		"Criteria":     criteria,
	}

	// Check what kind of update is requested
	target := c.Query("target")
	switch target {
	case "table":
		c.HTML(http.StatusOK, "job_list_partial.html", data)
	case "pagination":
		c.HTML(http.StatusOK, "pagination.html", data)
	default:
		// Return combined update with multiple targets
		c.HTML(http.StatusOK, "search_results.html", data)
	}
}

// EventStream handles server-sent events
func (h *Handler) EventStream(c *gin.Context) {
	if !h.config.SSEEnabled {
		c.String(http.StatusServiceUnavailable, "Server-sent events are disabled")
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Create SSE client
	client := h.broadcaster.AddClient(c)
	if client == nil {
		c.String(http.StatusServiceUnavailable, "Maximum SSE clients reached or SSE disabled")
		return
	}

	h.logger.WithField("client_id", client.id).Info("Starting SSE connection")

	// Serve the SSE connection using a simpler approach
	h.serveSSEConnection(c, client)
}

// serveSSEConnection handles SSE connection for a client with proper flushing
func (h *Handler) serveSSEConnection(c *gin.Context, client *SSEClient) {
	defer h.broadcaster.RemoveClient(client.id)

	// Send initial connection event
	h.writeSSEMessage(c, "connection", map[string]interface{}{
		"client_id": client.id,
		"connected": true,
	})

	// Send current job status
	h.sendCurrentJobStatus(c)

	// Handle events from the broadcaster
	for {
		select {
		case event, ok := <-client.events:
			if !ok {
				return
			}

			client.lastPing = time.Now()
			if !h.writeSSEMessage(c, string(event.Type), event.Data) {
				return
			}

		case <-client.ctx.Done():
			h.logger.WithField("client_id", client.id).Info("SSE client context cancelled")
			return

		case <-c.Request.Context().Done():
			h.logger.WithField("client_id", client.id).Info("SSE client request cancelled")
			return
		}
	}
}

// writeSSEMessage writes an SSE message to the client
func (h *Handler) writeSSEMessage(c *gin.Context, eventType string, data interface{}) bool {
	jsonData, err := json.Marshal(data)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal SSE event data")
		return false
	}

	// Write SSE event format
	message := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(jsonData))

	_, err = c.Writer.WriteString(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to write SSE message")
		return false
	}

	// Note: Flushing is handled automatically by Gin for streaming responses

	return true
}

// sendCurrentJobStatus sends the current status of all jobs to an SSE client
func (h *Handler) sendCurrentJobStatus(c *gin.Context) {
	jobs, err := h.jobStore.ListJobs(nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list jobs for SSE client")
		return
	}

	for _, job := range jobs {
		// Check if job is in failure state based on threshold
		isFailure := false
		if job.AutomaticFailureThreshold > 0 {
			timeSinceLastReport := time.Since(job.LastReportedAt)
			if timeSinceLastReport > time.Duration(job.AutomaticFailureThreshold)*time.Second {
				isFailure = true
			}
		}

		if !h.writeSSEMessage(c, "job-status-change", map[string]interface{}{
			"job_id":           job.ID,
			"name":             job.Name,
			"host":             job.Host,
			"status":           job.Status,
			"last_reported_at": job.LastReportedAt,
			"is_failure":       isFailure,
		}) {
			return
		}
	}
}

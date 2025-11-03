package dashboard

import (
	"encoding/json"
	"net/http"
	"strconv"

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
	logger       *logrus.Logger
}

// NewHandler creates a new dashboard handler
func NewHandler(config *config.DashboardConfig, jobStore *model.JobStore, logger *logrus.Logger) *Handler {
	return &Handler{
		config:       config,
		jobStore:     jobStore,
		assetHandler: NewAssetHandler(),
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
	jobs, err := h.jobStore.ListJobs(nil) // No label filters for now
	if err != nil {
		h.logger.WithError(err).Error("Failed to list jobs")
		c.String(http.StatusInternalServerError, "Failed to load jobs")
		return
	}

	data := gin.H{
		"Title":  h.config.Title,
		"Jobs":   jobs,
		"Config": h.config,
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

	// Return to job detail page
	c.Redirect(http.StatusFound, h.config.Path+"/jobs/"+strconv.Itoa(job.ID))
}

// JobSearch handles job search requests
func (h *Handler) JobSearch(c *gin.Context) {
	// TODO: Implement job search
	c.String(http.StatusNotImplemented, "Job search not implemented yet")
}

// EventStream handles server-sent events
func (h *Handler) EventStream(c *gin.Context) {
	// TODO: Implement server-sent events
	c.String(http.StatusNotImplemented, "Event stream not implemented yet")
}

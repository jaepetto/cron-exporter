package dashboard

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaepetto/cron-exporter/pkg/jobservice"
	"github.com/jaepetto/cron-exporter/pkg/model"
)

// APIError is the stable error response returned by dashboard JSON endpoints.
type APIError struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// DashboardRuntimeConfig contains non-secret values required by the portal.
type DashboardRuntimeConfig struct {
	BasePath        string `json:"base_path"`
	Title           string `json:"title"`
	PageSize        int    `json:"page_size"`
	SSEEnabled      bool   `json:"sse_enabled"`
	PollingFallback bool   `json:"polling_fallback"`
	PollingInterval int    `json:"polling_interval"`
}

// DashboardConfigAPI returns non-secret portal runtime configuration.
func (h *Handler) DashboardConfigAPI(c *gin.Context) {
	c.JSON(http.StatusOK, DashboardRuntimeConfig{
		BasePath:        h.config.Path,
		Title:           h.config.Title,
		PageSize:        h.config.PageSize,
		SSEEnabled:      h.config.SSEEnabled,
		PollingFallback: h.config.PollingFallback,
		PollingInterval: h.config.PollingInterval,
	})
}

// APIJobsList returns a filtered, paginated job collection.
func (h *Handler) APIJobsList(c *gin.Context) {
	criteria, fields := parseJobSearchCriteria(c)
	if len(fields) > 0 {
		h.writeAPIError(c, http.StatusBadRequest, "invalid_query", "invalid job search parameters", fields)
		return
	}

	result, err := h.jobService.List(criteria)
	if err != nil {
		h.writeJobServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// APIJobCreate creates a job from a JSON request.
func (h *Handler) APIJobCreate(c *gin.Context) {
	var input jobservice.CreateJobInput
	if !h.decodeJSON(c, &input) {
		return
	}

	job, err := h.jobService.Create(input)
	if err != nil {
		h.writeJobServiceError(c, err)
		return
	}
	h.broadcaster.BroadcastJobCreated(job)
	c.JSON(http.StatusCreated, job)
}

// APIJobDetail returns a job by ID.
func (h *Handler) APIJobDetail(c *gin.Context) {
	id, ok := h.parseJobID(c)
	if !ok {
		return
	}

	job, err := h.jobService.Get(id)
	if err != nil {
		h.writeJobServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, job)
}

// APIJobUpdate applies a partial JSON update to a job.
func (h *Handler) APIJobUpdate(c *gin.Context) {
	id, ok := h.parseJobID(c)
	if !ok {
		return
	}

	var input jobservice.UpdateJobInput
	if !h.decodeJSON(c, &input) {
		return
	}

	job, err := h.jobService.Update(id, input)
	if err != nil {
		h.writeJobServiceError(c, err)
		return
	}
	h.broadcaster.BroadcastJobUpdated(job)
	c.JSON(http.StatusOK, job)
}

// APIJobDelete removes a job by ID.
func (h *Handler) APIJobDelete(c *gin.Context) {
	id, ok := h.parseJobID(c)
	if !ok {
		return
	}

	job, err := h.jobService.Delete(id)
	if err != nil {
		h.writeJobServiceError(c, err)
		return
	}
	h.broadcaster.BroadcastJobDeleted(job.ID, job.Name, job.Host)
	c.Status(http.StatusNoContent)
}

// APIJobToggle toggles a job's maintenance status.
func (h *Handler) APIJobToggle(c *gin.Context) {
	id, ok := h.parseJobID(c)
	if !ok {
		return
	}

	job, err := h.jobService.ToggleMaintenance(id)
	if err != nil {
		h.writeJobServiceError(c, err)
		return
	}
	h.broadcaster.BroadcastJobStatusChange(job, isJobOverdue(job, time.Now()))
	c.JSON(http.StatusOK, job)
}

func (h *Handler) parseJobID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		h.writeAPIError(c, http.StatusBadRequest, "validation", "invalid job ID", map[string]string{"id": "must be a positive integer"})
		return 0, false
	}
	return id, true
}

func (h *Handler) decodeJSON(c *gin.Context, target interface{}) bool {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		h.writeAPIError(c, http.StatusBadRequest, "invalid_json", fmt.Sprintf("invalid JSON: %v", err), nil)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		h.writeAPIError(c, http.StatusBadRequest, "invalid_json", "request body must contain one JSON object", nil)
		return false
	}
	return true
}

func (h *Handler) writeJobServiceError(c *gin.Context, err error) {
	var serviceError *jobservice.Error
	if !errors.As(err, &serviceError) {
		h.logger.WithError(err).Error("Unhandled dashboard job error")
		h.writeAPIError(c, http.StatusInternalServerError, string(jobservice.ErrorInternal), "internal server error", nil)
		return
	}

	statusCode := http.StatusInternalServerError
	switch serviceError.Kind {
	case jobservice.ErrorValidation:
		statusCode = http.StatusBadRequest
	case jobservice.ErrorNotFound:
		statusCode = http.StatusNotFound
	case jobservice.ErrorConflict:
		statusCode = http.StatusConflict
	}
	if serviceError.Kind == jobservice.ErrorInternal {
		h.logger.WithError(err).Error(serviceError.Message)
	}
	h.writeAPIError(c, statusCode, string(serviceError.Kind), serviceError.Message, serviceError.Fields)
}

func (h *Handler) writeAPIError(c *gin.Context, statusCode int, code, message string, fields map[string]string) {
	c.JSON(statusCode, APIError{
		Code:      code,
		Message:   message,
		Fields:    fields,
		Timestamp: time.Now().UTC(),
	})
}

func parseJobSearchCriteria(c *gin.Context) (model.JobSearchCriteria, map[string]string) {
	criteria := model.JobSearchCriteria{
		Query:  strings.TrimSpace(c.Query("q")),
		Name:   strings.TrimSpace(c.Query("name")),
		Host:   strings.TrimSpace(c.Query("host")),
		Status: strings.TrimSpace(c.Query("status")),
		Labels: map[string]string{},
	}
	fields := map[string]string{}

	if value := c.Query("page"); value != "" {
		page, err := strconv.Atoi(value)
		if err != nil || page <= 0 {
			fields["page"] = "must be a positive integer"
		} else {
			criteria.Page = page
		}
	}
	if value := c.Query("page_size"); value != "" {
		pageSize, err := strconv.Atoi(value)
		if err != nil || pageSize <= 0 {
			fields["page_size"] = "must be a positive integer"
		} else {
			criteria.PageSize = pageSize
		}
	}
	if value := c.Query("before"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			fields["before"] = "must use RFC3339 format"
		} else {
			criteria.LastReportedBefore = &parsed
		}
	}
	if value := c.Query("after"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			fields["after"] = "must use RFC3339 format"
		} else {
			criteria.LastReportedAfter = &parsed
		}
	}
	for key, values := range c.Request.URL.Query() {
		if !strings.HasPrefix(key, "label.") || len(values) == 0 {
			continue
		}
		labelKey := strings.TrimSpace(strings.TrimPrefix(key, "label."))
		if labelKey == "" {
			fields["labels"] = "label keys cannot be empty"
			continue
		}
		criteria.Labels[labelKey] = values[0]
	}
	if len(criteria.Labels) == 0 {
		criteria.Labels = nil
	}

	return criteria, fields
}

func isJobOverdue(job *model.Job, now time.Time) bool {
	return job.AutomaticFailureThreshold > 0 && now.Sub(job.LastReportedAt) > time.Duration(job.AutomaticFailureThreshold)*time.Second
}

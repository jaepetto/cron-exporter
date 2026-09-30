package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaepetto/cron-exporter/pkg/config"
	"github.com/jaepetto/cron-exporter/pkg/jobservice"
	"github.com/jaepetto/cron-exporter/pkg/model"
	"github.com/sirupsen/logrus"
)

// Handler contains the HTTP behavior for the embedded portal.
type Handler struct {
	config      *config.DashboardConfig
	jobService  *jobservice.Service
	broadcaster *Broadcaster
	portal      *PortalHandler
	logger      *logrus.Logger
}

// NewHandler creates a dashboard handler.
func NewHandler(cfg *config.DashboardConfig, jobStore *model.JobStore, logger *logrus.Logger) *Handler {
	return &Handler{
		config:      cfg,
		jobService:  jobservice.New(jobStore),
		broadcaster: NewBroadcaster(cfg, logger),
		portal:      NewPortalHandler(cfg),
		logger:      logger,
	}
}

// ServePortal serves the React portal shell for a valid client-side route.
func (h *Handler) ServePortal(c *gin.Context) {
	h.portal.ServeIndex(c)
}

// ServePortalAsset serves a content-hashed React portal asset.
func (h *Handler) ServePortalAsset(c *gin.Context) {
	h.portal.ServeAsset(c)
}

// JobStatusAPI returns a job's calculated dashboard status.
func (h *Handler) JobStatusAPI(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		h.writeAPIError(c, http.StatusBadRequest, "validation", "invalid job ID", map[string]string{"id": "must be a positive integer"})
		return
	}

	job, err := h.jobService.Get(id)
	if err != nil {
		h.writeJobServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, JobStatusUpdate{
		JobID:          job.ID,
		Name:           job.Name,
		Host:           job.Host,
		Status:         job.Status,
		LastReportedAt: job.LastReportedAt,
		IsFailure:      isJobOverdue(job, time.Now()),
	})
}

// EventStream handles an authenticated server-sent event connection.
func (h *Handler) EventStream(c *gin.Context) {
	if !h.config.SSEEnabled {
		c.JSON(http.StatusServiceUnavailable, APIError{Code: "sse_disabled", Message: "server-sent events are disabled", Timestamp: time.Now().UTC()})
		return
	}

	client := h.broadcaster.AddClient(c)
	if client == nil {
		c.JSON(http.StatusServiceUnavailable, APIError{Code: "sse_unavailable", Message: "maximum SSE clients reached", Timestamp: time.Now().UTC()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	h.serveSSEConnection(c, client)
}

func (h *Handler) serveSSEConnection(c *gin.Context, client *SSEClient) {
	defer h.broadcaster.RemoveClient(client.id)

	connected := h.broadcaster.newEvent(EventConnected, nil)
	if !h.writeSSEMessage(c, string(connected.Type), connected) {
		return
	}

	for {
		select {
		case event, ok := <-client.events:
			if !ok || !h.writeSSEMessage(c, string(event.Type), event) {
				return
			}
		case <-client.ctx.Done():
			return
		case <-c.Request.Context().Done():
			return
		}
	}
}

func (h *Handler) writeSSEMessage(c *gin.Context, eventType string, data interface{}) bool {
	jsonData, err := json.Marshal(data)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal SSE event data")
		return false
	}

	eventID := ""
	if event, ok := data.(SSEEvent); ok && event.ID != "" {
		eventID = fmt.Sprintf("id: %s\n", event.ID)
	}
	if _, err := c.Writer.WriteString(fmt.Sprintf("%sevent: %s\ndata: %s\n\n", eventID, eventType, jsonData)); err != nil {
		h.logger.WithError(err).Error("Failed to write SSE message")
		return false
	}
	c.Writer.Flush()
	return true
}

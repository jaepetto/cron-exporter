package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jaepetto/cron-exporter/pkg/config"
)

// SetupRoutes configures all dashboard routes
func SetupRoutes(router *gin.Engine, config *config.DashboardConfig, handler *Handler, adminAPIKeys []string) {
	// Apply authentication middleware to all routes if required
	if config.AuthRequired {
		router.Use(AuthMiddlewareWithKeys(adminAPIKeys))
	}

	// Static assets
	router.GET("/assets/*filepath", handler.ServeAssets)

	// Main dashboard pages
	router.GET("/", handler.RedirectToDashboard)
	router.GET("/jobs", handler.JobsList)
	router.GET("/jobs/new", handler.JobCreateForm)
	router.POST("/jobs", handler.JobCreate)
	router.GET("/jobs/:id", handler.JobDetail)
	router.GET("/jobs/:id/edit", handler.JobEditForm)
	router.PUT("/jobs/:id", handler.JobUpdate)
	router.DELETE("/jobs/:id", handler.JobDelete)

	// HTMX endpoints for dynamic updates
	router.GET("/api/jobs", handler.JobsListAPI)
	router.GET("/api/jobs/:id/status", handler.JobStatusAPI)
	router.POST("/jobs/:id/toggle", handler.JobToggle)
	router.GET("/jobs/search", handler.JobSearch)

	// Server-sent events for real-time updates
	router.GET("/events", handler.EventStream)
}

// RedirectToDashboard redirects root dashboard path to jobs list
func (h *Handler) RedirectToDashboard(c *gin.Context) {
	c.Redirect(http.StatusFound, h.config.Path+"/jobs")
}

package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jaepetto/cron-exporter/pkg/config"
)

// SetupRoutes configures all dashboard routes
func SetupRoutes(router *gin.Engine, config *config.DashboardConfig, handler *Handler, adminAPIKeys []string) {
	// Static assets (no authentication required)
	router.GET("/assets/*filepath", handler.ServeAssets)

	// Create protected route group for authenticated routes
	var protectedRoutes gin.IRoutes = router
	if config.AuthRequired {
		authGroup := router.Group("/")
		authGroup.Use(AuthMiddlewareWithKeys(adminAPIKeys))
		protectedRoutes = authGroup
	}

	// Main dashboard pages (protected)
	protectedRoutes.GET("/", handler.RedirectToDashboard)
	protectedRoutes.GET("/jobs", handler.JobsList)
	protectedRoutes.GET("/jobs/new", handler.JobCreateForm)
	protectedRoutes.POST("/jobs", handler.JobCreate)
	protectedRoutes.GET("/jobs/:id", handler.JobDetail)
	protectedRoutes.GET("/jobs/:id/edit", handler.JobEditForm)
	protectedRoutes.PUT("/jobs/:id", handler.JobUpdate)  // For API usage
	protectedRoutes.POST("/jobs/:id", handler.JobUpdate) // For HTML forms
	protectedRoutes.DELETE("/jobs/:id", handler.JobDelete)
	protectedRoutes.POST("/jobs/:id/delete", handler.JobDelete) // For HTML delete forms

	// HTMX endpoints for dynamic updates (protected)
	protectedRoutes.GET("/api/jobs", handler.JobsListAPI)
	protectedRoutes.GET("/api/jobs/:id/status", handler.JobStatusAPI)
	protectedRoutes.POST("/jobs/:id/toggle", handler.JobToggle)
	protectedRoutes.GET("/jobs/search", handler.JobSearch)

	// Server-sent events for real-time updates (protected)
	protectedRoutes.GET("/events", handler.EventStream)
}

// RedirectToDashboard redirects root dashboard path to jobs list
func (h *Handler) RedirectToDashboard(c *gin.Context) {
	// Redirect to the full dashboard jobs path
	c.Redirect(http.StatusFound, h.config.Path+"/jobs")
}

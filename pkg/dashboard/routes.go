package dashboard

import (
	"github.com/gin-gonic/gin"
	"github.com/jaepetto/cron-exporter/pkg/config"
)

// SetupRoutes configures all dashboard routes
func SetupRoutes(router *gin.Engine, config *config.DashboardConfig, handler *Handler, adminAPIKeys []string) {
	// Content-hashed static assets contain no credentials and may be cached publicly.
	router.GET("/assets/*filepath", handler.ServePortalAsset)

	// Create protected route group for authenticated routes
	var protectedRoutes gin.IRoutes = router
	if config.AuthRequired {
		authGroup := router.Group("/")
		authGroup.Use(AuthMiddlewareWithKeys(adminAPIKeys))
		protectedRoutes = authGroup
	}

	// Portal shell routes (protected); API and asset paths are never handled as SPA fallbacks.
	protectedRoutes.GET("/", handler.ServePortal)
	protectedRoutes.GET("/jobs", handler.ServePortal)
	protectedRoutes.GET("/jobs/new", handler.ServePortal)
	protectedRoutes.GET("/jobs/:id", handler.ServePortal)
	protectedRoutes.GET("/jobs/:id/edit", handler.ServePortal)

	// Portal JSON API (protected)
	protectedRoutes.GET("/api/config", handler.DashboardConfigAPI)
	protectedRoutes.GET("/api/jobs", handler.APIJobsList)
	protectedRoutes.POST("/api/jobs", handler.APIJobCreate)
	protectedRoutes.GET("/api/jobs/:id", handler.APIJobDetail)
	protectedRoutes.PUT("/api/jobs/:id", handler.APIJobUpdate)
	protectedRoutes.DELETE("/api/jobs/:id", handler.APIJobDelete)
	protectedRoutes.POST("/api/jobs/:id/toggle", handler.APIJobToggle)
	protectedRoutes.GET("/api/jobs/:id/status", handler.JobStatusAPI)

	// Server-sent events for real-time updates (protected)
	protectedRoutes.GET("/events", handler.EventStream)
}

package dashboard

import (
	"github.com/gin-gonic/gin"
	"github.com/jaepetto/cron-exporter/pkg/config"
	"github.com/jaepetto/cron-exporter/pkg/model"
	"github.com/sirupsen/logrus"
)

// Dashboard represents the dashboard service
type Dashboard struct {
	config  *config.DashboardConfig
	handler *Handler
	router  *gin.Engine
	logger  *logrus.Logger
}

// New creates a new dashboard instance
func New(cfg *config.DashboardConfig, jobStore *model.JobStore, adminAPIKeys []string, logger *logrus.Logger) *Dashboard {
	// Set Gin mode based on config
	gin.SetMode(gin.ReleaseMode)

	// Create router
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(SecurityHeadersMiddleware())

	// Set up HTML templates using Gin's template renderer
	router.SetHTMLTemplate(LoadTemplates())

	// Create handler
	handler := NewHandler(cfg, jobStore, logger)

	// Setup routes
	SetupRoutes(router, cfg, handler, adminAPIKeys)

	return &Dashboard{
		config:  cfg,
		handler: handler,
		router:  router,
		logger:  logger,
	}
}

// Router returns the Gin router for mounting in the main server
func (d *Dashboard) Router() *gin.Engine {
	return d.router
}

// IsEnabled returns whether the dashboard is enabled
func (d *Dashboard) IsEnabled() bool {
	return d.config.Enabled
}

// GetBroadcaster returns the broadcaster for external use
func (d *Dashboard) GetBroadcaster() *Broadcaster {
	if d.handler == nil {
		return nil
	}
	return d.handler.broadcaster
}

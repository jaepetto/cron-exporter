package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaepetto/cron-exporter/pkg/api"
	"github.com/jaepetto/cron-exporter/pkg/metrics"
	"github.com/jaepetto/cron-exporter/pkg/model"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long: `Start the HTTP server to handle job result submissions
and serve Prometheus metrics.

The server provides:
- REST API for job CRUD operations
- Job result submission endpoint
- Prometheus metrics endpoint
- Health check endpoints`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runServer(); err != nil {
			logrus.WithError(err).Fatal("server failed")
		}
	},
}

func runServer() error {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"host": cfg.Server.Host,
		"port": cfg.Server.Port,
		"dev":  dev,
	}).Info("starting server")

	// Initialize database
	db, err := model.NewDatabase(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Configure database connection pool
	sqlxDB := db.GetDB()
	sqlxDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlxDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlxDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)

	// Create stores
	jobStore := model.NewJobStore(sqlxDB)
	jobResultStore := model.NewJobResultStore(sqlxDB)

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector(jobStore, jobResultStore)
	if err := metricsCollector.Register(); err != nil {
		return fmt.Errorf("failed to register metrics collector: %w", err)
	}

	// Create API server
	apiServer := api.NewServer(cfg, jobStore, jobResultStore, metricsCollector)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      apiServer.Handler(),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Start server in goroutine
	go func() {
		logrus.WithField("addr", server.Addr).Info("server listening")

		var err error
		if cfg.Security.RequireHTTPS {
			err = server.ListenAndServeTLS(cfg.Security.TLSCertFile, cfg.Security.TLSKeyFile)
		} else {
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("server failed to start")
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	logrus.Info("server exited")
	return nil
}

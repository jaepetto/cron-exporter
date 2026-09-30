package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/jaepetto/cron-exporter/pkg/config"
	"github.com/jaepetto/cron-exporter/pkg/dashboard"
	"github.com/jaepetto/cron-exporter/pkg/metrics"
	"github.com/jaepetto/cron-exporter/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestDashboardMountsAtConfiguredPath(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "api-dashboard-test.db")
	database, err := model.NewDatabase(databasePath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	jobStore := model.NewJobStore(database.GetDB())
	jobResultStore := model.NewJobResultStore(database.GetDB())
	collector := metrics.NewCollector(jobStore, jobResultStore)
	require.NoError(t, collector.Register())
	cfg := &config.Config{
		Database: config.DatabaseConfig{Path: databasePath},
		Metrics:  config.MetricsConfig{Path: "/metrics"},
		Security: config.SecurityConfig{AdminAPIKeys: []string{"dashboard-key"}},
		Dashboard: config.DashboardConfig{
			Enabled:         true,
			Path:            "/monitor",
			Title:           "Test Monitor",
			PageSize:        25,
			AuthRequired:    true,
			SSEEnabled:      true,
			SSETimeout:      30,
			SSEHeartbeat:    10,
			SSEMaxClients:   2,
			PollingFallback: true,
			PollingInterval: 5,
		},
	}
	server := NewServer(cfg, jobStore, jobResultStore, collector)
	t.Cleanup(func() { server.dashboard.GetBroadcaster().Stop() })

	unauthenticated := httptest.NewRecorder()
	server.Handler().ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/monitor/api/config", nil))
	require.Equal(t, http.StatusUnauthorized, unauthenticated.Code)

	authenticatedRequest := httptest.NewRequest(http.MethodGet, "/monitor/api/config", nil)
	authenticatedRequest.SetBasicAuth("admin", "dashboard-key")
	authenticated := httptest.NewRecorder()
	server.Handler().ServeHTTP(authenticated, authenticatedRequest)
	require.Equal(t, http.StatusOK, authenticated.Code, authenticated.Body.String())
	var runtimeConfig dashboard.DashboardRuntimeConfig
	require.NoError(t, json.Unmarshal(authenticated.Body.Bytes(), &runtimeConfig))
	require.Equal(t, "/monitor", runtimeConfig.BasePath)

	legacyPath := httptest.NewRecorder()
	server.Handler().ServeHTTP(legacyPath, httptest.NewRequest(http.MethodGet, "/dashboard/", nil))
	require.Equal(t, http.StatusNotFound, legacyPath.Code)
}

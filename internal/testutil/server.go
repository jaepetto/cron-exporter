package testutil

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/jaep/cron-exporter/pkg/api"
	"github.com/jaep/cron-exporter/pkg/config"
	"github.com/jaep/cron-exporter/pkg/metrics"
	"github.com/stretchr/testify/require"
)

// TestServer provides utilities for creating and managing test HTTP servers
type TestServer struct {
	Server   *httptest.Server
	Config   *config.Config
	Database *TestDatabase
	t        *testing.T
}

// NewTestServer creates a new test HTTP server with a test database
func NewTestServer(t *testing.T) *TestServer {
	// Create test database
	testDB := NewInMemoryTestDatabase(t)

	// Create test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         0, // Will be set by httptest.Server
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  120,
		},
		Database: config.DatabaseConfig{
			Path:            ":memory:",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 300,
		},
		Metrics: config.MetricsConfig{
			Path: "/metrics",
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Security: config.SecurityConfig{
			RequireHTTPS: false,
			APIKeys:      []string{"test-api-key"},
			AdminAPIKeys: []string{"admin-api-key"},
			TLSCertFile:  "",
			TLSKeyFile:   "",
		},
	}

	// Create stores
	jobStore := testDB.GetJobStore()
	jobResultStore := testDB.GetJobResultStore()

	// Create metrics collector
	metricsCollector := metrics.NewCollector(jobStore, jobResultStore)
	err := metricsCollector.Register()
	require.NoError(t, err, "Failed to register metrics collector")

	// Create API server
	apiServer := api.NewServer(cfg, jobStore, jobResultStore, metricsCollector)

	// Create HTTP test server
	server := httptest.NewServer(apiServer.Handler())

	return &TestServer{
		Server:   server,
		Config:   cfg,
		Database: testDB,
		t:        t,
	}
}

// NewTestServerWithAuth creates a test server with authentication enabled
func NewTestServerWithAuth(t *testing.T, adminAPIKeys []string, jobAPIKeys []string) *TestServer {
	testServer := NewTestServer(t)

	// Override security configuration
	testServer.Config.Security.AdminAPIKeys = adminAPIKeys
	testServer.Config.Security.APIKeys = jobAPIKeys

	// Use a non-dev database path to enable auth
	testServer.Config.Database.Path = "/tmp/test_cronmetrics.db"

	return testServer
}

// Close closes the test server and cleans up resources
func (ts *TestServer) Close() {
	if ts.Server != nil {
		ts.Server.Close()
	}
	if ts.Database != nil {
		ts.Database.Close()
	}
}

// URL returns the base URL of the test server
func (ts *TestServer) URL() string {
	return ts.Server.URL
}

// AdminHeaders returns HTTP headers with admin API key for authenticated requests
func (ts *TestServer) AdminHeaders() map[string]string {
	if len(ts.Config.Security.AdminAPIKeys) > 0 {
		return map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", ts.Config.Security.AdminAPIKeys[0]),
			"Content-Type":  "application/json",
		}
	}
	return map[string]string{
		"Content-Type": "application/json",
	}
}

// JobHeaders returns HTTP headers with job API key for job result submissions
func (ts *TestServer) JobHeaders() map[string]string {
	if len(ts.Config.Security.APIKeys) > 0 {
		return map[string]string{
			"X-API-Key":    ts.Config.Security.APIKeys[0],
			"Content-Type": "application/json",
		}
	}
	return map[string]string{
		"Content-Type": "application/json",
	}
}

// SeedTestData adds test data to the server's database
func (ts *TestServer) SeedTestData() {
	ts.Database.SeedTestData()
}

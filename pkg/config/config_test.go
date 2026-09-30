package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validTestConfig() *Config {
	return &Config{
		Server: ServerConfig{Port: 8080},
		Database: DatabaseConfig{
			Path: "/tmp/cronmetrics-test.db",
		},
		Metrics:  MetricsConfig{Path: "/metrics"},
		Logging:  LoggingConfig{Level: "info", Format: "json"},
		Security: SecurityConfig{RequireHTTPS: false},
		Dashboard: DashboardConfig{
			Enabled:         true,
			Path:            "/dashboard",
			Title:           "Cron Monitor",
			RefreshInterval: 5,
			PageSize:        25,
			SSEEnabled:      true,
			SSETimeout:      300,
			SSEHeartbeat:    30,
			SSEMaxClients:   100,
			PollingFallback: true,
			PollingInterval: 5,
		},
	}
}

func TestValidateConfigRejectsReservedDashboardPaths(t *testing.T) {
	for _, dashboardPath := range []string{"/", "/api", "/api/dashboard", "/metrics", "/health", "/swagger"} {
		t.Run(dashboardPath, func(t *testing.T) {
			cfg := validTestConfig()
			cfg.Dashboard.Path = dashboardPath

			require.ErrorContains(t, validateConfig(cfg), "dashboard path")
		})
	}
}

func TestValidateConfigRejectsNonCanonicalDashboardPaths(t *testing.T) {
	for _, dashboardPath := range []string{"dashboard", "/dashboard/", "/dashboard/../portal"} {
		t.Run(dashboardPath, func(t *testing.T) {
			cfg := validTestConfig()
			cfg.Dashboard.Path = dashboardPath

			require.ErrorContains(t, validateConfig(cfg), "dashboard path")
		})
	}
}

func TestValidateConfigRejectsInvalidDashboardRealtimeSettings(t *testing.T) {
	tests := map[string]func(*DashboardConfig){
		"timeout":   func(cfg *DashboardConfig) { cfg.SSETimeout = 0 },
		"heartbeat": func(cfg *DashboardConfig) { cfg.SSEHeartbeat = 0 },
		"heartbeat before timeout": func(cfg *DashboardConfig) {
			cfg.SSEHeartbeat = cfg.SSETimeout
		},
		"client limit":     func(cfg *DashboardConfig) { cfg.SSEMaxClients = 0 },
		"polling interval": func(cfg *DashboardConfig) { cfg.PollingInterval = 0 },
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := validTestConfig()
			mutate(&cfg.Dashboard)

			require.Error(t, validateConfig(cfg))
		})
	}
}

func TestValidateConfigAcceptsDisabledRealtimeFeatures(t *testing.T) {
	cfg := validTestConfig()
	cfg.Dashboard.SSEEnabled = false
	cfg.Dashboard.SSETimeout = 0
	cfg.Dashboard.SSEHeartbeat = 0
	cfg.Dashboard.SSEMaxClients = 0
	cfg.Dashboard.PollingFallback = false
	cfg.Dashboard.PollingInterval = 0

	require.NoError(t, validateConfig(cfg))
}

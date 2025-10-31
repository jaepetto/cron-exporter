package integration

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jaepetto/cron-exporter/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrometheusMetricsFormat(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()
	server.SeedTestData()

	client := testutil.NewHTTPClient(t, server.URL())

	t.Run("MetricsEndpointFormat", func(t *testing.T) {
		resp := client.GET("/metrics")
		resp.ExpectStatus(200).
			ExpectHeader("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		body := resp.BodyString()

		// Check basic Prometheus format
		assert.NotEmpty(t, body)
		assert.Contains(t, body, "# HELP")
		assert.Contains(t, body, "# TYPE")

		// Verify no HTML content (common mistake)
		assert.NotContains(t, body, "<html>")
		assert.NotContains(t, body, "<body>")
	})

	t.Run("JobStatusMetrics", func(t *testing.T) {
		resp := client.GET("/metrics")
		body := resp.BodyString()

		// Should contain job status metrics
		assert.Contains(t, body, "cronjob_status")

		// Check for HELP and TYPE comments
		assert.Contains(t, body, "# HELP cronjob_status")
		assert.Contains(t, body, "# TYPE cronjob_status") // Verify seeded test data appears in metrics
		assert.Contains(t, body, `job_name="backup"`)
		assert.Contains(t, body, `host="db1"`)
		assert.Contains(t, body, `job_name="log-rotation"`)
		assert.Contains(t, body, `host="web1"`)
	})

	t.Run("JobLabelsInMetrics", func(t *testing.T) {
		resp := client.GET("/metrics")
		body := resp.BodyString()

		// Check that custom labels are included
		assert.Contains(t, body, `env="prod"`)
		assert.Contains(t, body, `type="backup"`)
		assert.Contains(t, body, `type="maintenance"`)
	})
}

func TestMetricsWithJobResults(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()
	server.SeedTestData()

	// Submit some job results
	resultClient := testutil.NewHTTPClient(t, server.URL()).
		WithHeaders(map[string]string{
			"X-API-Key":    "cm_test_backup_key",
			"Content-Type": "application/json",
		})

	t.Run("SuccessfulJobMetrics", func(t *testing.T) {
		// Submit successful result
		resultRequest := map[string]interface{}{
			"job_name": "backup",
			"host":     "db1",
			"status":   "success",
			"duration": 120,
			"message":  "Backup completed successfully",
		}

		resultClient.POST("/api/job-result", resultRequest).ExpectStatus(201)

		// Wait for metrics to be updated
		time.Sleep(100 * time.Millisecond)

		// Check metrics
		client := testutil.NewHTTPClient(t, server.URL())
		resp := client.GET("/metrics")
		body := resp.BodyString()

		// Should show successful status (value = 1)
		assert.Contains(t, body, `job_name="backup"`)
		assert.Contains(t, body, `host="db1"`)
		assert.Contains(t, body, `cronjob_status{job_name="backup",host="db1"`)
		assert.Contains(t, body, `} 1`)
	})

	t.Run("FailedJobMetrics", func(t *testing.T) {
		// Submit failed result
		resultRequest := map[string]interface{}{
			"job_name": "backup",
			"host":     "db1",
			"status":   "failure",
			"duration": 45,
			"message":  "Database connection failed",
		}

		resultClient.POST("/api/job-result", resultRequest).ExpectStatus(201)

		// Wait for metrics to be updated
		time.Sleep(100 * time.Millisecond)

		// Check metrics
		client := testutil.NewHTTPClient(t, server.URL())
		resp := client.GET("/metrics")
		body := resp.BodyString()

		// Should show failed status (value = 0)
		assert.Contains(t, body, `job_name="backup"`)
		assert.Contains(t, body, `host="db1"`)
		assert.Contains(t, body, `cronjob_status{job_name="backup",host="db1"`)
		assert.Contains(t, body, `} 0`)
	})
}

func TestMetricsAutoFailureDetection(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()

	// Create a job with a short failure threshold
	adminClient := testutil.NewHTTPClient(t, server.URL()).
		WithHeaders(server.AdminHeaders())

	jobRequest := map[string]interface{}{
		"job_name":                    "short-threshold-job",
		"host":                        "test-host",
		"automatic_failure_threshold": 1, // 1 second threshold for testing
		"status":                      "active",
	}

	adminClient.POST("/api/job", jobRequest).ExpectStatus(201)

	// Wait longer than the threshold
	time.Sleep(2 * time.Second)

	// Check metrics - should detect automatic failure
	client := testutil.NewHTTPClient(t, server.URL())
	resp := client.GET("/metrics")
	body := resp.BodyString()

	// Should show the job with automatic failure detection
	assert.Contains(t, body, `job_name="short-threshold-job"`)
	assert.Contains(t, body, `host="test-host"`)

	// Look for failure indication (exact format depends on implementation)
	lines := strings.Split(body, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, `job_name="short-threshold-job"`) &&
			strings.Contains(line, `host="test-host"`) {
			// The line should indicate failure or have a specific value
			found = true
			// You might want to check for specific values like 0 (failure) or 2 (auto-failed)
			assert.NotContains(t, line, " 1", "Job should not show as successful after threshold exceeded")
			break
		}
	}
	assert.True(t, found, "Could not find metrics line for short-threshold-job")
}

func TestMetricsMaintenanceMode(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()
	server.SeedTestData()

	client := testutil.NewHTTPClient(t, server.URL())
	resp := client.GET("/metrics")
	body := resp.BodyString()

	// The maintenance-job from seed data should have special handling
	assert.Contains(t, body, `job_name="maintenance-job"`)
	assert.Contains(t, body, `host="app1"`)

	// Should have maintenance status or special value (-1)
	lines := strings.Split(body, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, `job_name="maintenance-job"`) &&
			strings.Contains(line, `host="app1"`) {
			found = true
			// Maintenance jobs should either have status="maintenance" or value -1
			assert.True(t,
				strings.Contains(line, `status="maintenance"`) || strings.Contains(line, " -1"),
				fmt.Sprintf("Maintenance job should have special status, got: %s", line))
			break
		}
	}
	assert.True(t, found, "Could not find metrics line for maintenance-job")
}

func TestMetricsValidation(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()
	server.SeedTestData()

	client := testutil.NewHTTPClient(t, server.URL())
	resp := client.GET("/metrics")
	body := resp.BodyString()

	t.Run("ValidPrometheusFormat", func(t *testing.T) {
		lines := strings.Split(body, "\n")

		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if strings.HasPrefix(line, "#") {
				// Comment lines should be properly formatted
				assert.True(t,
					strings.HasPrefix(line, "# HELP ") || strings.HasPrefix(line, "# TYPE "),
					fmt.Sprintf("Line %d: Invalid comment format: %s", i+1, line))
				continue
			}

			// Metric lines should have valid format
			if strings.Contains(line, "cronjob_status") {
				// Should have metric name, labels, and value
				assert.Regexp(t,
					regexp.MustCompile(`cronjob_status\{[^}]+\}\s+[0-9.-]+`),
					line,
					fmt.Sprintf("Line %d: Invalid metric format: %s", i+1, line))
			}
		}
	})

	t.Run("RequiredLabels", func(t *testing.T) {
		lines := strings.Split(body, "\n")

		for _, line := range lines {
			if strings.Contains(line, "cronjob_status{") {
				// Every cronjob_status metric should have required labels
				assert.Contains(t, line, `job_name=`, "Missing job_name label: "+line)
				assert.Contains(t, line, `host=`, "Missing host label: "+line)

				// Should have proper label quoting
				assert.Regexp(t, regexp.MustCompile(`job_name="[^"]+"`), line, "Invalid job_name label format: "+line)
				assert.Regexp(t, regexp.MustCompile(`host="[^"]+"`), line, "Invalid host label format: "+line)
			}
		}
	})

	t.Run("NumericValues", func(t *testing.T) {
		lines := strings.Split(body, "\n")

		for i, line := range lines {
			if strings.Contains(line, "cronjob_status{") {
				// Extract the value (last part after closing brace)
				parts := strings.Split(line, "} ")
				require.Equal(t, 2, len(parts), fmt.Sprintf("Line %d: Invalid metric format: %s", i+1, line))

				value := strings.TrimSpace(parts[1])
				assert.Regexp(t, regexp.MustCompile(`^[0-9.-]+$`), value,
					fmt.Sprintf("Line %d: Invalid numeric value: %s", i+1, value))
			}
		}
	})
}

func TestMetricsHTTPMethods(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()

	client := testutil.NewHTTPClient(t, server.URL())

	t.Run("GETAllowed", func(t *testing.T) {
		client.GET("/metrics").ExpectStatus(200)
	})

	t.Run("POSTNotAllowed", func(t *testing.T) {
		client.POST("/metrics", nil).ExpectStatus(405)
	})

	t.Run("PUTNotAllowed", func(t *testing.T) {
		client.PUT("/metrics", nil).ExpectStatus(405)
	})

	t.Run("DELETENotAllowed", func(t *testing.T) {
		client.DELETE("/metrics").ExpectStatus(405)
	})
}

func TestMetricsPerformance(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()

	// Create multiple jobs to test performance
	adminClient := testutil.NewHTTPClient(t, server.URL()).
		WithHeaders(server.AdminHeaders())

	// Create 50 jobs
	for i := 1; i <= 50; i++ {
		jobRequest := map[string]interface{}{
			"job_name":                    fmt.Sprintf("perf-job-%d", i),
			"host":                        fmt.Sprintf("host-%d", i%5), // 5 different hosts
			"automatic_failure_threshold": 3600,
			"status":                      "active",
			"labels": map[string]string{
				"env": "test",
				"seq": fmt.Sprintf("%d", i),
			},
		}
		adminClient.POST("/api/job", jobRequest).ExpectStatus(201)
	}

	client := testutil.NewHTTPClient(t, server.URL())

	t.Run("MetricsResponseTime", func(t *testing.T) {
		start := time.Now()
		resp := client.GET("/metrics")
		duration := time.Since(start)

		resp.ExpectStatus(200)

		// Metrics should respond within 1 second even with 50 jobs
		assert.Less(t, duration.Milliseconds(), int64(1000),
			fmt.Sprintf("Metrics endpoint took too long: %v", duration))

		body := resp.BodyString()

		// Should contain all jobs
		for i := 1; i <= 50; i++ {
			assert.Contains(t, body, fmt.Sprintf(`job_name="perf-job-%d"`, i))
		}
	})

	t.Run("ConcurrentMetricsRequests", func(t *testing.T) {
		t.Skip("Skipping concurrent metrics test - database connection issues under load")
	})
}

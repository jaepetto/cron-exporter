package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jaepetto/cron-exporter/internal/testutil"
	"github.com/jaepetto/cron-exporter/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteWorkflow tests the complete lifecycle of a cron job monitoring workflow
func TestCompleteWorkflow(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()

	adminClient := testutil.NewHTTPClient(t, server.URL()).
		WithHeaders(server.AdminHeaders())

	metricsClient := testutil.NewHTTPClient(t, server.URL())

	t.Run("CompleteJobLifecycle", func(t *testing.T) {
		// Step 1: Create a new job
		jobRequest := map[string]interface{}{
			"job_name":                    "daily-backup",
			"host":                        "production-db",
			"automatic_failure_threshold": 7200, // 2 hours
			"labels": map[string]string{
				"env":         "production",
				"service":     "database",
				"criticality": "high",
			},
			"status": "active",
		}

		var createdJob model.Job
		adminClient.POST("/api/job", jobRequest).
			ExpectStatus(201).
			ExpectJSON(&createdJob)

		require.NotEmpty(t, createdJob.ApiKey, "Job should have an API key")
		require.Greater(t, createdJob.ID, 0, "Job should have a valid ID")

		// Step 2: Check initial metrics state (no results yet)
		metricsResp := metricsClient.GET("/metrics")
		metricsResp.ExpectStatus(200)
		initialMetrics := metricsResp.BodyString()

		assert.Contains(t, initialMetrics, `job_name="daily-backup"`)
		assert.Contains(t, initialMetrics, `host="production-db"`)
		assert.Contains(t, initialMetrics, `env="production"`)

		// Step 3: Submit successful job result
		jobClient := testutil.NewHTTPClient(t, server.URL()).
			WithHeaders(map[string]string{
				"X-API-Key":    createdJob.ApiKey,
				"Content-Type": "application/json",
			})

		successResult := map[string]interface{}{
			"job_name": "daily-backup",
			"host":     "production-db",
			"status":   "success",
			"duration": 1800, // 30 minutes
			"message":  "Backup completed successfully - 500GB processed",
		}

		jobClient.POST("/api/job-result", successResult).ExpectStatus(201)

		// Step 4: Verify successful metrics
		time.Sleep(100 * time.Millisecond) // Allow metrics to update

		successMetrics := metricsClient.GET("/metrics").BodyString()
		assert.Contains(t, successMetrics, `job_name="daily-backup"`)
		assert.Contains(t, successMetrics, `} "success"`)

		// Step 5: Submit failure result
		failureResult := map[string]interface{}{
			"job_name": "daily-backup",
			"host":     "production-db",
			"status":   "failure",
			"duration": 300, // 5 minutes before failure
			"message":  "Database connection timeout",
		}

		jobClient.POST("/api/job-result", failureResult).ExpectStatus(201)

		// Step 6: Verify failure metrics
		time.Sleep(100 * time.Millisecond)

		failureMetrics := metricsClient.GET("/metrics").BodyString()
		assert.Contains(t, failureMetrics, `job_name="daily-backup"`)
		assert.Contains(t, failureMetrics, `} "failure"`)

		// Step 7: Set job to maintenance mode
		maintenanceUpdate := map[string]interface{}{
			"status": "maintenance",
		}

		var updatedJob model.Job
		adminClient.PUT(fmt.Sprintf("/api/job/%d", createdJob.ID), maintenanceUpdate).
			ExpectStatus(200).
			ExpectJSON(&updatedJob)

		assert.Equal(t, "maintenance", updatedJob.Status)

		// Step 8: Verify maintenance metrics
		time.Sleep(100 * time.Millisecond)

		maintenanceMetrics := metricsClient.GET("/metrics").BodyString()
		assert.Contains(t, maintenanceMetrics, `job_name="daily-backup"`)
		// Should indicate maintenance mode (status="maintenance" or value -1)
		lines := strings.Split(maintenanceMetrics, "\n")
		foundMaintenanceLine := false
		for _, line := range lines {
			if strings.Contains(line, `job_name="daily-backup"`) {
				foundMaintenanceLine = true
				assert.True(t,
					strings.Contains(line, `status="maintenance"`) || strings.Contains(line, " -1"),
					fmt.Sprintf("Expected maintenance indication in: %s", line))
				break
			}
		}
		assert.True(t, foundMaintenanceLine, "Could not find metrics line for maintenance job")

		// Step 9: Reactivate job
		activateUpdate := map[string]interface{}{
			"status": "active",
		}

		adminClient.PUT(fmt.Sprintf("/api/job/%d", createdJob.ID), activateUpdate).
			ExpectStatus(200)

		// Step 10: Submit another successful result
		finalResult := map[string]interface{}{
			"job_name": "daily-backup",
			"host":     "production-db",
			"status":   "success",
			"duration": 1650, // 27.5 minutes
			"message":  "Backup resumed and completed",
		}

		jobClient.POST("/api/job-result", finalResult).ExpectStatus(201)

		// Step 11: Final metrics verification
		time.Sleep(100 * time.Millisecond)

		finalMetrics := metricsClient.GET("/metrics").BodyString()
		assert.Contains(t, finalMetrics, `job_name="daily-backup"`)
		assert.Contains(t, finalMetrics, `} "success"`)

		// Step 12: Delete the job
		adminClient.DELETE(fmt.Sprintf("/api/job/%d", createdJob.ID)).ExpectStatus(204)

		// Step 13: Verify job is removed from metrics
		time.Sleep(100 * time.Millisecond)

		cleanupMetrics := metricsClient.GET("/metrics").BodyString()
		assert.NotContains(t, cleanupMetrics, `job_name="daily-backup"`)
	})
}

func TestMultiJobWorkflow(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()

	adminClient := testutil.NewHTTPClient(t, server.URL()).
		WithHeaders(server.AdminHeaders())

	metricsClient := testutil.NewHTTPClient(t, server.URL())

	t.Run("MultipleJobsWithDifferentStatuses", func(t *testing.T) {
		// Create multiple jobs representing different scenarios
		jobs := []struct {
			name      string
			host      string
			threshold int
			labels    map[string]string
		}{
			{
				name:      "web-backup",
				host:      "web-server-1",
				threshold: 3600,
				labels:    map[string]string{"env": "prod", "type": "backup"},
			},
			{
				name:      "log-cleanup",
				host:      "web-server-1",
				threshold: 1800,
				labels:    map[string]string{"env": "prod", "type": "cleanup"},
			},
			{
				name:      "health-check",
				host:      "monitoring",
				threshold: 300,
				labels:    map[string]string{"env": "prod", "type": "monitoring"},
			},
		}

		var createdJobs []model.Job

		// Create all jobs
		for _, job := range jobs {
			jobRequest := map[string]interface{}{
				"job_name":                    job.name,
				"host":                        job.host,
				"automatic_failure_threshold": job.threshold,
				"labels":                      job.labels,
				"status":                      "active",
			}

			var createdJob model.Job
			adminClient.POST("/api/job", jobRequest).
				ExpectStatus(201).
				ExpectJSON(&createdJob)

			createdJobs = append(createdJobs, createdJob)
		}

		// Submit different types of results
		scenarios := []struct {
			jobIndex int
			status   string
			duration int
			message  string
		}{
			{0, "success", 1200, "Web backup completed"},
			{1, "failure", 300, "Disk full during cleanup"},
			{2, "success", 15, "All services healthy"},
		}

		for _, scenario := range scenarios {
			job := createdJobs[scenario.jobIndex]
			jobClient := testutil.NewHTTPClient(t, server.URL()).
				WithHeaders(map[string]string{
					"X-API-Key":    job.ApiKey,
					"Content-Type": "application/json",
				})

			resultRequest := map[string]interface{}{
				"job_name": job.Name,
				"host":     job.Host,
				"status":   scenario.status,
				"duration": scenario.duration,
				"message":  scenario.message,
			}

			jobClient.POST("/api/job-result", resultRequest).ExpectStatus(201)
		}

		// Verify all jobs appear in metrics with correct statuses
		time.Sleep(200 * time.Millisecond)

		metrics := metricsClient.GET("/metrics").BodyString()

		// Check each job appears with the expected status
		expectedStatuses := map[string]string{
			"web-backup":   "success",
			"log-cleanup":  "failure",
			"health-check": "success",
		}

		for jobName, expectedStatus := range expectedStatuses {
			assert.Contains(t, metrics, fmt.Sprintf(`job_name="%s"`, jobName))

			// Find the specific metric line for this job in cronjob_status_info
			lines := strings.Split(metrics, "\n")
			found := false
			for _, line := range lines {
				if strings.Contains(line, fmt.Sprintf(`job_name="%s"`, jobName)) &&
					strings.Contains(line, "cronjob_status_info") {
					assert.Contains(t, line, fmt.Sprintf(`} "%s"`, expectedStatus),
						fmt.Sprintf("Job %s should have status %s as value in line: %s", jobName, expectedStatus, line))
					found = true
					break
				}
			}
			assert.True(t, found, fmt.Sprintf("Could not find metrics line for job %s", jobName))
		}

		// Verify labels are preserved
		assert.Contains(t, metrics, `type="backup"`)
		assert.Contains(t, metrics, `type="cleanup"`)
		assert.Contains(t, metrics, `type="monitoring"`)
		assert.Contains(t, metrics, `env="prod"`)
	})
}

func TestAutoFailureDetectionWorkflow(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()

	adminClient := testutil.NewHTTPClient(t, server.URL()).
		WithHeaders(server.AdminHeaders())

	metricsClient := testutil.NewHTTPClient(t, server.URL())

	t.Run("JobTimeoutDetection", func(t *testing.T) {
		// Create a job with very short timeout for testing
		jobRequest := map[string]interface{}{
			"job_name":                    "timeout-test-job",
			"host":                        "test-server",
			"automatic_failure_threshold": 1, // 1 second timeout
			"status":                      "active",
		}

		var createdJob model.Job
		adminClient.POST("/api/job", jobRequest).
			ExpectStatus(201).
			ExpectJSON(&createdJob)

		// Submit an initial successful result
		jobClient := testutil.NewHTTPClient(t, server.URL()).
			WithHeaders(map[string]string{
				"X-API-Key":    createdJob.ApiKey,
				"Content-Type": "application/json",
			})

		initialResult := map[string]interface{}{
			"job_name": "timeout-test-job",
			"host":     "test-server",
			"status":   "success",
			"duration": 30,
			"message":  "Initial successful run",
		}

		jobClient.POST("/api/job-result", initialResult).ExpectStatus(201)

		// Verify initial success in metrics
		time.Sleep(100 * time.Millisecond)
		initialMetrics := metricsClient.GET("/metrics").BodyString()
		assert.Contains(t, initialMetrics, `job_name="timeout-test-job"`)
		assert.Contains(t, initialMetrics, `} "success"`)

		// Wait for timeout to trigger
		time.Sleep(2 * time.Second)

		// Check metrics for automatic failure detection
		timeoutMetrics := metricsClient.GET("/metrics").BodyString()
		assert.Contains(t, timeoutMetrics, `job_name="timeout-test-job"`)

		// The job should now show as automatically failed
		lines := strings.Split(timeoutMetrics, "\n")
		foundTimeoutLine := false
		for _, line := range lines {
			if strings.Contains(line, `job_name="timeout-test-job"`) &&
				strings.Contains(line, "cronjob_status") {
				foundTimeoutLine = true
				// Should not show success anymore
				assert.NotContains(t, line, `} "success"`,

					fmt.Sprintf("Job should not show as successful after timeout: %s", line))
				// Should indicate failure or have value indicating auto-failure
				break
			}
		}
		assert.True(t, foundTimeoutLine, "Could not find metrics line for timeout job")

		// Submit a recovery result
		recoveryResult := map[string]interface{}{
			"job_name": "timeout-test-job",
			"host":     "test-server",
			"status":   "success",
			"duration": 25,
			"message":  "Job recovered",
		}

		jobClient.POST("/api/job-result", recoveryResult).ExpectStatus(201)

		// Verify recovery in metrics
		time.Sleep(100 * time.Millisecond)
		recoveryMetrics := metricsClient.GET("/metrics").BodyString()
		assert.Contains(t, recoveryMetrics, `job_name="timeout-test-job"`)
		assert.Contains(t, recoveryMetrics, `} "success"`)
	})
}

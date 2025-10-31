package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/jaep/cron-exporter/internal/testutil"
	"github.com/jaep/cron-exporter/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestAPIHealthCheck(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()

	client := testutil.NewHTTPClient(t, server.URL())

	// Test health endpoint
	var health map[string]interface{}
	client.GET("/health").
		ExpectStatus(200).
		ExpectHeader("Content-Type", "application/json").
		ExpectJSON(&health)

	assert.Equal(t, "healthy", health["status"])
	assert.Contains(t, health, "timestamp")
	assert.Contains(t, health, "version")
}

func TestJobCRUDOperations(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()

	client := testutil.NewHTTPClient(t, server.URL()).
		WithHeaders(server.AdminHeaders())

	t.Run("CreateJob", func(t *testing.T) {
		jobRequest := map[string]interface{}{
			"job_name":                    "test-backup",
			"host":                        "db1",
			"automatic_failure_threshold": 3600,
			"labels": map[string]string{
				"env":  "test",
				"type": "backup",
			},
			"status": "active",
		}

		var jobResponse model.Job
		client.POST("/api/job", jobRequest).
			ExpectStatus(201).
			ExpectJSON(&jobResponse)

		assert.Equal(t, "test-backup", jobResponse.Name)
		assert.Equal(t, "db1", jobResponse.Host)
		assert.Equal(t, 3600, jobResponse.AutomaticFailureThreshold)
		assert.Equal(t, "active", jobResponse.Status)
		assert.Equal(t, "test", jobResponse.Labels["env"])
		assert.Equal(t, "backup", jobResponse.Labels["type"])
		assert.NotEmpty(t, jobResponse.ApiKey)
		assert.Greater(t, jobResponse.ID, 0)
	})

	t.Run("ListJobs", func(t *testing.T) {
		// First create a few jobs
		for i := 1; i <= 3; i++ {
			jobRequest := map[string]interface{}{
				"job_name":                    fmt.Sprintf("job-%d", i),
				"host":                        "test-host",
				"automatic_failure_threshold": 1800,
				"status":                      "active",
			}
			client.POST("/api/job", jobRequest).ExpectStatus(201)
		}

		// List all jobs
		var jobs []model.Job
		client.GET("/api/job").
			ExpectStatus(200).
			ExpectJSON(&jobs)

		assert.GreaterOrEqual(t, len(jobs), 3)
	})

	t.Run("GetJobByID", func(t *testing.T) {
		// Create a job first
		jobRequest := map[string]interface{}{
			"job_name":                    "get-test-job",
			"host":                        "test-host",
			"automatic_failure_threshold": 1200,
			"status":                      "active",
		}

		var createdJob model.Job
		client.POST("/api/job", jobRequest).
			ExpectStatus(201).
			ExpectJSON(&createdJob)

		// Get the job by ID
		var retrievedJob model.Job
		client.GET(fmt.Sprintf("/api/job/%d", createdJob.ID)).
			ExpectStatus(200).
			ExpectJSON(&retrievedJob)

		assert.Equal(t, createdJob.ID, retrievedJob.ID)
		assert.Equal(t, createdJob.Name, retrievedJob.Name)
		assert.Equal(t, createdJob.Host, retrievedJob.Host)
	})

	t.Run("UpdateJobByID", func(t *testing.T) {
		// Create a job first
		jobRequest := map[string]interface{}{
			"job_name":                    "update-test-job",
			"host":                        "test-host",
			"automatic_failure_threshold": 1200,
			"status":                      "active",
		}

		var createdJob model.Job
		client.POST("/api/job", jobRequest).
			ExpectStatus(201).
			ExpectJSON(&createdJob)

		// Update the job
		updateRequest := map[string]interface{}{
			"automatic_failure_threshold": 2400,
			"status":                      "maintenance",
			"labels": map[string]string{
				"env": "production",
			},
		}

		var updatedJob model.Job
		client.PUT(fmt.Sprintf("/api/job/%d", createdJob.ID), updateRequest).
			ExpectStatus(200).
			ExpectJSON(&updatedJob)

		assert.Equal(t, createdJob.ID, updatedJob.ID)
		assert.Equal(t, 2400, updatedJob.AutomaticFailureThreshold)
		assert.Equal(t, "maintenance", updatedJob.Status)
		assert.Equal(t, "production", updatedJob.Labels["env"])
	})

	t.Run("DeleteJobByID", func(t *testing.T) {
		// Create a job first
		jobRequest := map[string]interface{}{
			"job_name":                    "delete-test-job",
			"host":                        "test-host",
			"automatic_failure_threshold": 1200,
			"status":                      "active",
		}

		var createdJob model.Job
		client.POST("/api/job", jobRequest).
			ExpectStatus(201).
			ExpectJSON(&createdJob)

		// Delete the job
		client.DELETE(fmt.Sprintf("/api/job/%d", createdJob.ID)).
			ExpectStatus(204)

		// Verify it's deleted
		client.GET(fmt.Sprintf("/api/job/%d", createdJob.ID)).
			ExpectStatus(404)
	})
}

func TestJobResultSubmission(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()
	server.SeedTestData()

	t.Run("SuccessfulJobResult", func(t *testing.T) {
		client := testutil.NewHTTPClient(t, server.URL()).
			WithHeaders(map[string]string{
				"X-API-Key":    "cm_test_backup_key",
				"Content-Type": "application/json",
			})

		resultRequest := map[string]interface{}{
			"job_name": "backup",
			"host":     "db1",
			"status":   "success",
			"duration": 120,
			"message":  "Backup completed successfully",
		}

		var response map[string]interface{}
		client.POST("/api/job-result", resultRequest).
			ExpectStatus(201).
			ExpectJSON(&response)

		assert.Equal(t, "recorded", response["status"])

		// Verify the result was stored
		assert.Greater(t, server.Database.CountJobResults(), 0)
	})

	t.Run("FailedJobResult", func(t *testing.T) {
		client := testutil.NewHTTPClient(t, server.URL()).
			WithHeaders(map[string]string{
				"X-API-Key":    "cm_test_backup_key",
				"Content-Type": "application/json",
			})

		resultRequest := map[string]interface{}{
			"job_name": "backup",
			"host":     "db1",
			"status":   "failure",
			"duration": 45,
			"message":  "Database connection failed",
		}

		var response map[string]interface{}
		client.POST("/api/job-result", resultRequest).
			ExpectStatus(201).
			ExpectJSON(&response)

		assert.Equal(t, "recorded", response["status"])
	})

	t.Run("InvalidJobResult", func(t *testing.T) {
		client := testutil.NewHTTPClient(t, server.URL()).
			WithHeaders(map[string]string{
				"X-API-Key":    "cm_test_backup_key",
				"Content-Type": "application/json",
			})

		resultRequest := map[string]interface{}{
			"job_name": "backup",
			"host":     "db1",
			"status":   "invalid-status",
		}

		client.POST("/api/job-result", resultRequest).
			ExpectStatus(400).
			ExpectContains("status must be 'success' or 'failure'")
	})

	t.Run("MissingJobResult", func(t *testing.T) {
		client := testutil.NewHTTPClient(t, server.URL()).
			WithHeaders(map[string]string{
				"X-API-Key":    "cm_test_backup_key",
				"Content-Type": "application/json",
			})

		resultRequest := map[string]interface{}{
			"host":   "db1",
			"status": "success",
		}

		client.POST("/api/job-result", resultRequest).
			ExpectStatus(400).
			ExpectContains("job_name, host, and status are required")
	})
}

func TestMetricsEndpoint(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()
	server.SeedTestData()

	client := testutil.NewHTTPClient(t, server.URL())

	// Submit some results to generate metrics
	resultClient := testutil.NewHTTPClient(t, server.URL()).
		WithHeaders(map[string]string{
			"X-API-Key":    "cm_test_backup_key",
			"Content-Type": "application/json",
		})

	// Submit a successful result
	resultRequest := map[string]interface{}{
		"job_name": "backup",
		"host":     "db1",
		"status":   "success",
		"duration": 120,
	}
	resultClient.POST("/api/job-result", resultRequest).ExpectStatus(201)

	// Give a moment for metrics to be updated
	time.Sleep(100 * time.Millisecond)

	// Test metrics endpoint
	resp := client.GET("/metrics").ExpectStatus(200)
	resp.ExpectHeader("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	body := resp.BodyString()

	// Check for basic Prometheus format
	assert.Contains(t, body, "# HELP")
	assert.Contains(t, body, "# TYPE")

	// Check for job-specific metrics
	assert.Contains(t, body, "cronjob_status")
	assert.Contains(t, body, "job_name=\"backup\"")
	assert.Contains(t, body, "host=\"db1\"")
}

func TestJobCRUDValidation(t *testing.T) {
	server := testutil.NewTestServer(t)
	defer server.Close()

	client := testutil.NewHTTPClient(t, server.URL()).
		WithHeaders(server.AdminHeaders())

	t.Run("CreateJobMissingName", func(t *testing.T) {
		jobRequest := map[string]interface{}{
			"host":                        "test-host",
			"automatic_failure_threshold": 3600,
		}

		client.POST("/api/job", jobRequest).
			ExpectStatus(400).
			ExpectContains("job name and host are required")
	})

	t.Run("CreateJobMissingHost", func(t *testing.T) {
		jobRequest := map[string]interface{}{
			"job_name":                    "test-job",
			"automatic_failure_threshold": 3600,
		}

		client.POST("/api/job", jobRequest).
			ExpectStatus(400).
			ExpectContains("job name and host are required")
	})

	t.Run("CreateJobWithNegativeThreshold", func(t *testing.T) {
		jobRequest := map[string]interface{}{
			"job_name":                    "test-job",
			"host":                        "test-host",
			"automatic_failure_threshold": -1,
		}

		// API currently allows negative thresholds
		var job map[string]interface{}
		client.POST("/api/job", jobRequest).ExpectStatus(201).ExpectJSON(&job)
		assert.Equal(t, -1, int(job["automatic_failure_threshold"].(float64)))
	})

	t.Run("CreateJobWithCustomStatus", func(t *testing.T) {
		jobRequest := map[string]interface{}{
			"job_name":                    "invalid-status-job",
			"host":                        "test-host",
			"automatic_failure_threshold": 3600,
			"status":                      "invalid",
		}

		// API currently allows any status value
		var job map[string]interface{}
		client.POST("/api/job", jobRequest).ExpectStatus(201).ExpectJSON(&job)
		assert.Equal(t, "invalid", job["status"])
	})
}

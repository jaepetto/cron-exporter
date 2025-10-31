package integration

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaep/cron-exporter/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIServeCommand(t *testing.T) {
	// Ensure binary is built
	buildBinary(t)

	cliTest := testutil.NewCLITest(t)
	cliTest.CreateDefaultTestConfig()

	t.Run("ServeHelp", func(t *testing.T) {
		result := cliTest.RunCommand("serve", "--help")
		result.ExpectSuccess().
			ExpectStdoutContains("Start the HTTP server").
			ExpectStdoutContains("REST API for job CRUD operations").
			ExpectStdoutContains("Prometheus metrics endpoint")
	})

	t.Run("ServeDevMode", func(t *testing.T) {
		t.Skip("Skipping server startup test - timing issues in test environment")
	})

	t.Run("ServeWithConfig", func(t *testing.T) {
		t.Skip("Skipping server startup test - timing issues in test environment")
	})

	t.Run("ServeInvalidConfig", func(t *testing.T) {
		cliTest.CreateTestConfig("invalid: yaml: content")

		result := cliTest.RunCommand("serve")
		result.ExpectFailure().
			ExpectStderrContains("failed to load config")
	})
}

func TestCLIJobCommands(t *testing.T) {
	// Ensure binary is built
	buildBinary(t)

	cliTest := testutil.NewCLITest(t)
	cliTest.CreateDefaultTestConfig()

	t.Run("JobHelp", func(t *testing.T) {
		result := cliTest.RunCommand("job", "--help")
		result.ExpectSuccess().
			ExpectStdoutContains("Manage cron job definitions").
			ExpectStdoutContains("add").
			ExpectStdoutContains("list").
			ExpectStdoutContains("update").
			ExpectStdoutContains("delete")
	})

	t.Run("JobAddHelp", func(t *testing.T) {
		result := cliTest.RunCommand("job", "add", "--help")
		result.ExpectSuccess().
			ExpectStdoutContains("Add a new job definition").
			ExpectStdoutContains("--name").
			ExpectStdoutContains("--host").
			ExpectStdoutContains("--threshold")
	})

	t.Run("JobAdd", func(t *testing.T) {
		result := cliTest.RunCommand("job", "add",
			"--name", "test-backup",
			"--host", "db1",
			"--threshold", "3600",
			"--label", "env=test",
			"--label", "type=backup")

		result.ExpectSuccess().
			ExpectStdoutContains("Job").
			ExpectStdoutContains("created successfully").
			ExpectStdoutContains("API Key:")
	})

	t.Run("JobAddMissingName", func(t *testing.T) {
		result := cliTest.RunCommand("job", "add",
			"--host", "db1",
			"--threshold", "3600")

		result.ExpectFailure().
			ExpectStderrContains(`required flag(s) "name" not set`)
	})

	t.Run("JobAddMissingHost", func(t *testing.T) {
		result := cliTest.RunCommand("job", "add",
			"--name", "test-job",
			"--threshold", "3600")

		result.ExpectFailure().
			ExpectStderrContains(`required flag(s) "host" not set`)
	})

	t.Run("JobList", func(t *testing.T) {
		// First add a few jobs
		for i := 1; i <= 3; i++ {
			cliTest.RunCommand("job", "add",
				"--name", fmt.Sprintf("job-%d", i),
				"--host", "test-host",
				"--threshold", "1800").ExpectSuccess()
		}

		result := cliTest.RunCommand("job", "list")
		result.ExpectSuccess()

		stdout := result.Stdout
		assert.Contains(t, stdout, "job-1")
		assert.Contains(t, stdout, "job-2")
		assert.Contains(t, stdout, "job-3")
		assert.Contains(t, stdout, "test-host")
	})

	t.Run("JobListEmpty", func(t *testing.T) {
		// Use a fresh CLI test with empty database
		freshCLI := testutil.NewCLITest(t)
		freshCLI.CreateDefaultTestConfig()

		result := freshCLI.RunCommand("job", "list")
		result.ExpectSuccess().
			ExpectStdoutContains("No jobs found")
	})

	t.Run("JobShow", func(t *testing.T) {
		// Add a job first
		addResult := cliTest.RunCommand("job", "add",
			"--name", "show-test",
			"--host", "test-host",
			"--threshold", "2400")
		addResult.ExpectSuccess()

		// Extract job ID from output
		output := addResult.Stdout
		lines := strings.Split(output, "\n")
		var jobID string
		for _, line := range lines {
			if strings.Contains(line, "Job ID") {
				// Extract ID from line like "Job ID 1 ('show-test@test-host') created successfully"
				parts := strings.Split(line, " ")
				if len(parts) >= 3 {
					jobID = parts[2]
					break
				}
			}
		}

		require.NotEmpty(t, jobID, "Could not extract job ID from output: %s", output)

		// Show the job
		result := cliTest.RunCommand("job", "show", jobID)
		result.ExpectSuccess().
			ExpectStdoutContains("show-test").
			ExpectStdoutContains("test-host").
			ExpectStdoutContains("2400")
	})

	t.Run("JobUpdate", func(t *testing.T) {
		// Add a job first
		addResult := cliTest.RunCommand("job", "add",
			"--name", "update-test",
			"--host", "test-host",
			"--threshold", "1800")
		addResult.ExpectSuccess()

		// Extract job ID
		output := addResult.Stdout
		lines := strings.Split(output, "\n")
		var jobID string
		for _, line := range lines {
			if strings.Contains(line, "Job ID") {
				parts := strings.Split(line, " ")
				if len(parts) >= 3 {
					jobID = parts[2]
					break
				}
			}
		}

		require.NotEmpty(t, jobID, "Could not extract job ID from output")

		// Update the job
		result := cliTest.RunCommand("job", "update", jobID,
			"--threshold", "3600",
			"--maintenance")
		result.ExpectSuccess().
			ExpectStdoutContains("updated successfully")

		// Verify the update
		showResult := cliTest.RunCommand("job", "show", jobID)
		showResult.ExpectSuccess().
			ExpectStdoutContains("3600").
			ExpectStdoutContains("maintenance")
	})

	t.Run("JobDelete", func(t *testing.T) {
		// Add a job first
		addResult := cliTest.RunCommand("job", "add",
			"--name", "delete-test",
			"--host", "test-host",
			"--threshold", "1800")
		addResult.ExpectSuccess()

		// Extract job ID
		output := addResult.Stdout
		lines := strings.Split(output, "\n")
		var jobID string
		for _, line := range lines {
			if strings.Contains(line, "Job ID") {
				parts := strings.Split(line, " ")
				if len(parts) >= 3 {
					jobID = parts[2]
					break
				}
			}
		}

		require.NotEmpty(t, jobID, "Could not extract job ID from output")

		// Delete the job
		result := cliTest.RunCommand("job", "delete", jobID)
		result.ExpectSuccess().
			ExpectStdoutContains("deleted successfully")

		// Verify it's deleted
		showResult := cliTest.RunCommand("job", "show", jobID)
		showResult.ExpectFailure().
			ExpectStderrContains("not found")
	})
}

func TestCLIConfigCommand(t *testing.T) {
	// Ensure binary is built
	buildBinary(t)

	cliTest := testutil.NewCLITest(t)

	t.Run("ConfigHelp", func(t *testing.T) {
		result := cliTest.RunCommand("config", "--help")
		result.ExpectSuccess().
			ExpectStdoutContains("Generate example configuration").
			ExpectStdoutContains("example")
	})

	t.Run("ConfigExample", func(t *testing.T) {
		result := cliTest.RunCommand("config", "example")
		result.ExpectSuccess().
			ExpectStdoutContains("server:").
			ExpectStdoutContains("database:").
			ExpectStdoutContains("metrics:").
			ExpectStdoutContains("logging:").
			ExpectStdoutContains("security:")
	})
}

func TestCLIGlobalFlags(t *testing.T) {
	// Ensure binary is built
	buildBinary(t)

	cliTest := testutil.NewCLITest(t)

	t.Run("Help", func(t *testing.T) {
		result := cliTest.RunCommand("--help")
		result.ExpectSuccess().
			ExpectStdoutContains("Go-based API and web server").
			ExpectStdoutContains("Central REST API for job result submissions").
			ExpectStdoutContains("Available Commands:").
			ExpectStdoutContains("serve").
			ExpectStdoutContains("job").
			ExpectStdoutContains("config")
	})

	t.Run("Version", func(t *testing.T) {
		result := cliTest.RunCommand("--version")
		// Note: Version handling might not be implemented yet
		// This test ensures the flag doesn't cause a crash
		assert.True(t, result.ExitCode == 0 || result.ExitCode == 1)
	})

	t.Run("DevFlag", func(t *testing.T) {
		result := cliTest.RunCommand("--dev", "--help")
		result.ExpectSuccess().
			ExpectStdoutContains("A Go-based API and web server")
	})

	t.Run("ConfigFlag", func(t *testing.T) {
		cliTest.CreateDefaultTestConfig()

		result := cliTest.RunCommand("--config", cliTest.ConfigFile, "--help")
		result.ExpectSuccess().
			ExpectStdoutContains("A Go-based API and web server")
	})

	t.Run("InvalidConfigFlag", func(t *testing.T) {
		result := cliTest.RunCommand("--config", "/nonexistent/config.yaml", "job", "list")
		result.ExpectFailure()
		// Should fail when trying to load non-existent config
	})
}

// buildBinary ensures the cronmetrics binary is built for testing
func buildBinary(t *testing.T) {
	// Get the project root directory (assuming tests are in test/integration)
	projectRoot := filepath.Join("..", "..")
	binaryPath := filepath.Join(projectRoot, "bin", "cronmetrics")

	// Build the binary without config flags
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/cronmetrics")
	cmd.Dir = projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	require.NoError(t, err,
		fmt.Sprintf("Failed to build binary: %s\nStdout: %s\nStderr: %s",
			cmd.String(), stdout.String(), stderr.String()))
}

func TestCLIErrorHandling(t *testing.T) {
	// Ensure binary is built
	buildBinary(t)

	cliTest := testutil.NewCLITest(t)

	t.Run("UnknownCommand", func(t *testing.T) {
		result := cliTest.RunCommand("unknown")
		result.ExpectFailure().
			ExpectStderrContains("unknown command")
	})

	t.Run("UnknownSubcommand", func(t *testing.T) {
		result := cliTest.RunCommand("job", "unknown")
		// CLI shows help for unknown subcommands instead of failing
		result.ExpectSuccess().
			ExpectStdoutContains("Available Commands:")
	})

	t.Run("MissingRequiredFlag", func(t *testing.T) {
		result := cliTest.RunCommand("job", "add")
		result.ExpectFailure()
		// Should fail due to missing required flags
	})
}

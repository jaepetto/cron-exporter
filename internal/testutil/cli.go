package testutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// CLITest provides utilities for testing command-line interface
type CLITest struct {
	TempDir    string
	ConfigFile string
	DBFile     string
	BinaryPath string
	Env        []string
	t          *testing.T
}

// NewCLITest creates a new CLI test environment
func NewCLITest(t *testing.T) *CLITest {
	tempDir := t.TempDir()

	// Get absolute path to binary - look from current working directory up to project root
	var binaryPath string
	wd, _ := os.Getwd()

	// Try different paths relative to the test location
	candidates := []string{
		filepath.Join(wd, "../../bin/cronmetrics"),
		filepath.Join(wd, "../bin/cronmetrics"),
		filepath.Join(wd, "bin/cronmetrics"),
		"/Users/jaep/code/ic/cron-exporter/bin/cronmetrics", // Fallback absolute path
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			binaryPath = candidate
			break
		}
	}

	if binaryPath == "" {
		t.Fatalf("Could not find cronmetrics binary in any expected location")
	}

	return &CLITest{
		TempDir:    tempDir,
		ConfigFile: filepath.Join(tempDir, "config.yaml"),
		DBFile:     filepath.Join(tempDir, "test.db"),
		BinaryPath: binaryPath,
		Env:        os.Environ(),
		t:          t,
	}
}

// WithBinaryPath sets a custom path to the cronmetrics binary
func (c *CLITest) WithBinaryPath(path string) *CLITest {
	c.BinaryPath = path
	return c
}

// WithEnv adds environment variables
func (c *CLITest) WithEnv(key, value string) *CLITest {
	c.Env = append(c.Env, fmt.Sprintf("%s=%s", key, value))
	return c
}

// CreateTestConfig creates a test configuration file
func (c *CLITest) CreateTestConfig(config string) {
	err := os.WriteFile(c.ConfigFile, []byte(config), 0644)
	require.NoError(c.t, err, "Failed to create test config file")
}

// CreateDefaultTestConfig creates a default test configuration
func (c *CLITest) CreateDefaultTestConfig() {
	config := fmt.Sprintf(`
server:
  host: "localhost"
  port: 8080
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 120

database:
  path: "%s"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 300

metrics:
  path: "/metrics"

logging:
  level: "info"
  format: "text"
  output: "stdout"

security:
  require_https: false
  api_keys:
    - "test-api-key"
  admin_api_keys:
    - "admin-api-key"
`, c.DBFile)

	c.CreateTestConfig(config)
}

// RunCommand executes a cronmetrics command and returns the result
func (c *CLITest) RunCommand(args ...string) *CLIResult {
	return c.RunCommandWithTimeout(30*time.Second, args...)
}

// RunCommandWithTimeout executes a cronmetrics command with a timeout
func (c *CLITest) RunCommandWithTimeout(timeout time.Duration, args ...string) *CLIResult {
	// Add config file flag if not already specified
	hasConfig := false
	for _, arg := range args {
		if arg == "--config" {
			hasConfig = true
			break
		}
	}

	if !hasConfig && c.ConfigFile != "" {
		args = append([]string{"--config", c.ConfigFile}, args...)
	}

	cmd := exec.Command(c.BinaryPath, args...)
	cmd.Env = c.Env
	cmd.Dir = c.TempDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Create a channel to signal completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		return &CLIResult{
			Command:  fmt.Sprintf("%s %s", c.BinaryPath, strings.Join(args, " ")),
			ExitCode: cmd.ProcessState.ExitCode(),
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			Error:    err,
			t:        c.t,
		}
	case <-time.After(timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		require.Fail(c.t, fmt.Sprintf("Command timed out after %v: %s %s", timeout, c.BinaryPath, strings.Join(args, " ")))
		return nil
	}
}

// RunBackground starts a command in the background and returns immediately
func (c *CLITest) RunBackground(args ...string) *BackgroundProcess {
	// Add config file flag if not already specified
	hasConfig := false
	for _, arg := range args {
		if arg == "--config" {
			hasConfig = true
			break
		}
	}

	if !hasConfig && c.ConfigFile != "" {
		args = append([]string{"--config", c.ConfigFile}, args...)
	}

	cmd := exec.Command(c.BinaryPath, args...)
	cmd.Env = c.Env
	cmd.Dir = c.TempDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Start()
	require.NoError(c.t, err, fmt.Sprintf("Failed to start background command: %s %s", c.BinaryPath, strings.Join(args, " ")))

	return &BackgroundProcess{
		Command: fmt.Sprintf("%s %s", c.BinaryPath, strings.Join(args, " ")),
		Process: cmd.Process,
		Cmd:     cmd,
		Stdout:  &stdout,
		Stderr:  &stderr,
		t:       c.t,
	}
}

// CLIResult represents the result of a CLI command execution
type CLIResult struct {
	Command  string
	ExitCode int
	Stdout   string
	Stderr   string
	Error    error
	t        *testing.T
}

// ExpectSuccess asserts that the command executed successfully
func (r *CLIResult) ExpectSuccess() *CLIResult {
	require.NoError(r.t, r.Error, fmt.Sprintf("Command failed: %s\nStdout: %s\nStderr: %s", r.Command, r.Stdout, r.Stderr))
	require.Equal(r.t, 0, r.ExitCode, fmt.Sprintf("Expected exit code 0, got %d for command: %s\nStdout: %s\nStderr: %s", r.ExitCode, r.Command, r.Stdout, r.Stderr))
	return r
}

// ExpectFailure asserts that the command failed
func (r *CLIResult) ExpectFailure() *CLIResult {
	require.NotEqual(r.t, 0, r.ExitCode, fmt.Sprintf("Expected command to fail, but it succeeded: %s\nStdout: %s", r.Command, r.Stdout))
	return r
}

// ExpectExitCode asserts that the command exited with the expected code
func (r *CLIResult) ExpectExitCode(expected int) *CLIResult {
	require.Equal(r.t, expected, r.ExitCode, fmt.Sprintf("Expected exit code %d, got %d for command: %s\nStdout: %s\nStderr: %s", expected, r.ExitCode, r.Command, r.Stdout, r.Stderr))
	return r
}

// ExpectStdoutContains asserts that stdout contains the expected string
func (r *CLIResult) ExpectStdoutContains(expected string) *CLIResult {
	require.Contains(r.t, r.Stdout, expected, fmt.Sprintf("Expected stdout to contain '%s' for command: %s\nStdout: %s", expected, r.Command, r.Stdout))
	return r
}

// ExpectStderrContains asserts that stderr contains the expected string
func (r *CLIResult) ExpectStderrContains(expected string) *CLIResult {
	require.Contains(r.t, r.Stderr, expected, fmt.Sprintf("Expected stderr to contain '%s' for command: %s\nStderr: %s", expected, r.Command, r.Stderr))
	return r
}

// BackgroundProcess represents a process running in the background
type BackgroundProcess struct {
	Command string
	Process *os.Process
	Cmd     *exec.Cmd
	Stdout  *bytes.Buffer
	Stderr  *bytes.Buffer
	t       *testing.T
}

// Stop stops the background process
func (bp *BackgroundProcess) Stop() {
	if bp.Process != nil {
		bp.Process.Kill()
		bp.Cmd.Wait() // Wait for the process to exit
	}
}

// Wait waits for the background process to complete
func (bp *BackgroundProcess) Wait() *CLIResult {
	err := bp.Cmd.Wait()
	return &CLIResult{
		Command:  bp.Command,
		ExitCode: bp.Cmd.ProcessState.ExitCode(),
		Stdout:   bp.Stdout.String(),
		Stderr:   bp.Stderr.String(),
		Error:    err,
		t:        bp.t,
	}
}

// GetOutput returns the current output from the background process
func (bp *BackgroundProcess) GetOutput() (stdout, stderr string) {
	return bp.Stdout.String(), bp.Stderr.String()
}

// WaitForOutput waits for the specified string to appear in stdout
func (bp *BackgroundProcess) WaitForOutput(expected string, timeout time.Duration) bool {
	start := time.Now()
	for time.Since(start) < timeout {
		if strings.Contains(bp.Stdout.String(), expected) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

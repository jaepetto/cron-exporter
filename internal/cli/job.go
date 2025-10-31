package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jaep/cron-exporter/pkg/model"
	"github.com/jaep/cron-exporter/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// jobCmd represents the job command
var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "Job management operations",
	Long: `Manage cron job definitions including create, list, update, and delete operations.

Jobs are identified by the combination of name and host, allowing multiple
jobs with the same name to run on different hosts.`,
}

func init() {
	jobCmd.AddCommand(jobAddCmd)
	jobCmd.AddCommand(jobListCmd)
	jobCmd.AddCommand(jobUpdateCmd)
	jobCmd.AddCommand(jobDeleteCmd)
	jobCmd.AddCommand(jobShowCmd)
}

// jobAddCmd adds a new job
var jobAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new job",
	Long:  `Add a new job definition with specified name, host, and configuration`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runJobAdd(cmd); err != nil {
			logrus.WithError(err).Fatal("failed to add job")
		}
	},
}

var (
	jobID        int
	jobName      string
	jobHost      string
	jobApiKey    string
	jobThreshold int
	jobLabels    []string
	jobStatus    string
)

func init() {
	jobAddCmd.Flags().StringVarP(&jobName, "name", "n", "", "job name (required)")
	jobAddCmd.Flags().StringVar(&jobHost, "host", "", "host name (required)")
	jobAddCmd.Flags().StringVar(&jobApiKey, "api-key", "", "API key for the job (auto-generated if not provided)")
	jobAddCmd.Flags().IntVarP(&jobThreshold, "threshold", "t", 3600, "automatic failure threshold in seconds")
	jobAddCmd.Flags().StringSliceVarP(&jobLabels, "label", "l", []string{}, "labels in key=value format")
	jobAddCmd.Flags().StringVarP(&jobStatus, "status", "s", "active", "job status (active, maintenance, paused)")

	jobAddCmd.MarkFlagRequired("name")
	jobAddCmd.MarkFlagRequired("host")
}

func runJobAdd(cmd *cobra.Command) error {
	if jobName == "" || jobHost == "" {
		return fmt.Errorf("job name and host are required")
	}

	// Parse labels
	labels, err := parseLabels(jobLabels)
	if err != nil {
		return fmt.Errorf("invalid labels: %w", err)
	}

	// Generate API key if not provided
	apiKey := jobApiKey
	if apiKey == "" {
		generated, err := util.GenerateAPIKey()
		if err != nil {
			return fmt.Errorf("failed to generate API key: %w", err)
		}
		apiKey = generated
	}

	// Load configuration and initialize database
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	db, err := model.NewDatabase(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	jobStore := model.NewJobStore(db.GetDB())

	// Create job
	job := &model.Job{
		Name:                      jobName,
		Host:                      jobHost,
		ApiKey:                    apiKey,
		AutomaticFailureThreshold: jobThreshold,
		Labels:                    labels,
		Status:                    jobStatus,
		LastReportedAt:            time.Now().UTC(),
	}

	if err := jobStore.CreateJob(job); err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	fmt.Printf("Job ID %d ('%s@%s') created successfully\n", job.ID, jobName, jobHost)
	fmt.Printf("API Key: %s\n", apiKey)

	if jobApiKey == "" {
		fmt.Println("\nNOTE: Save this API key for your cron jobs to submit results.")
		fmt.Printf("You can retrieve it later using: cronmetrics job show %d\n", job.ID)
	}

	return nil
}

// jobListCmd lists jobs
var jobListCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs",
	Long:  `List all jobs with optional filtering by labels`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runJobList(cmd); err != nil {
			logrus.WithError(err).Fatal("failed to list jobs")
		}
	},
}

var (
	listLabels  []string
	outputJSON  bool
	showApiKeys bool
)

func init() {
	jobListCmd.Flags().StringSliceVarP(&listLabels, "label", "l", []string{}, "filter by labels in key=value format")
	jobListCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "output as JSON")
	jobListCmd.Flags().BoolVar(&showApiKeys, "show-api-keys", false, "show API keys (masked for security)")
}

func runJobList(cmd *cobra.Command) error {
	// Parse label filters
	labelFilters, err := parseLabels(listLabels)
	if err != nil {
		return fmt.Errorf("invalid label filters: %w", err)
	}

	// Load configuration and initialize database
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	db, err := model.NewDatabase(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	jobStore := model.NewJobStore(db.GetDB())

	// List jobs
	jobs, err := jobStore.ListJobs(labelFilters)
	if err != nil {
		return fmt.Errorf("failed to list jobs: %w", err)
	}

	if outputJSON {
		output, err := json.MarshalIndent(jobs, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		printJobsTable(jobs)
	}

	return nil
}

// jobUpdateCmd updates a job
var jobUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a job",
	Long:  `Update an existing job's configuration by ID`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runJobUpdate(cmd, args); err != nil {
			logrus.WithError(err).Fatal("failed to update job")
		}
	},
}

var (
	updateThreshold *int
	updateLabels    []string
	updateStatus    string
	maintenance     bool
)

func init() {
	jobUpdateCmd.Flags().StringVarP(&jobName, "name", "n", "", "update job name")
	jobUpdateCmd.Flags().StringVar(&jobHost, "host", "", "update host name")
	jobUpdateCmd.Flags().StringVar(&jobApiKey, "api-key", "", "update API key for the job")
	jobUpdateCmd.Flags().IntVar(&jobThreshold, "threshold", 0, "automatic failure threshold in seconds")
	jobUpdateCmd.Flags().StringSliceVarP(&updateLabels, "label", "l", []string{}, "labels in key=value format")
	jobUpdateCmd.Flags().StringVarP(&updateStatus, "status", "s", "", "job status (active, maintenance, paused)")
	jobUpdateCmd.Flags().BoolVarP(&maintenance, "maintenance", "m", false, "set job to maintenance mode")
}

func runJobUpdate(cmd *cobra.Command, args []string) error {
	// Parse job ID from argument
	jobID, err := parseJobID(args[0])
	if err != nil {
		return fmt.Errorf("invalid job ID: %w", err)
	}

	// Load configuration and initialize database
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	db, err := model.NewDatabase(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	jobStore := model.NewJobStore(db.GetDB())

	// Get existing job
	job, err := jobStore.GetJobByID(jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Update fields if provided
	if cmd.Flags().Changed("name") {
		job.Name = jobName
	}

	if cmd.Flags().Changed("host") {
		job.Host = jobHost
	}

	if cmd.Flags().Changed("api-key") {
		job.ApiKey = jobApiKey
	}

	if cmd.Flags().Changed("threshold") {
		job.AutomaticFailureThreshold = jobThreshold
	}

	if len(updateLabels) > 0 {
		labels, err := parseLabels(updateLabels)
		if err != nil {
			return fmt.Errorf("invalid labels: %w", err)
		}
		job.Labels = labels
	}

	if updateStatus != "" {
		job.Status = updateStatus
	}

	if maintenance {
		job.Status = "maintenance"
	}

	// Update job
	if err := jobStore.UpdateJobByID(job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	fmt.Printf("Job ID %d ('%s@%s') updated successfully\n", job.ID, job.Name, job.Host)
	return nil
}

// jobDeleteCmd deletes a job
var jobDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a job",
	Long:  `Delete a job definition by ID`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runJobDelete(cmd, args); err != nil {
			logrus.WithError(err).Fatal("failed to delete job")
		}
	},
}

func runJobDelete(cmd *cobra.Command, args []string) error {
	// Parse job ID from argument
	jobID, err := parseJobID(args[0])
	if err != nil {
		return fmt.Errorf("invalid job ID: %w", err)
	}

	// Load configuration and initialize database
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	db, err := model.NewDatabase(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	jobStore := model.NewJobStore(db.GetDB())

	// Get job info before deleting (for display purposes)
	job, err := jobStore.GetJobByID(jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Delete job
	if err := jobStore.DeleteJobByID(jobID); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	fmt.Printf("Job ID %d ('%s@%s') deleted successfully\n", job.ID, job.Name, job.Host)
	return nil
}

// jobShowCmd shows detailed job information
var jobShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show job details",
	Long:  `Show detailed information about a specific job by ID`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runJobShow(cmd, args); err != nil {
			logrus.WithError(err).Fatal("failed to show job")
		}
	},
}

func init() {
	jobShowCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "output as JSON")
}

func runJobShow(cmd *cobra.Command, args []string) error {
	// Parse job ID from argument
	jobID, err := parseJobID(args[0])
	if err != nil {
		return fmt.Errorf("invalid job ID: %w", err)
	}

	// Load configuration and initialize database
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	db, err := model.NewDatabase(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	jobStore := model.NewJobStore(db.GetDB())

	// Get job by ID
	job, err := jobStore.GetJobByID(jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	if outputJSON {
		output, err := json.MarshalIndent(job, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))
	} else {
		printJobDetails(job)
	}

	return nil
}

// parseLabels parses key=value label strings into a map
func parseLabels(labelStings []string) (map[string]string, error) {
	labels := make(map[string]string)
	for _, label := range labelStings {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid label format: %s (expected key=value)", label)
		}
		labels[parts[0]] = parts[1]
	}
	return labels, nil
}

// printJobsTable prints jobs in table format
func printJobsTable(jobs []*model.Job) {
	if len(jobs) == 0 {
		fmt.Println("No jobs found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if showApiKeys {
		fmt.Fprintln(w, "ID\tNAME\tHOST\tAPI_KEY\tSTATUS\tTHRESHOLD\tLAST_REPORTED\tLABELS")
	} else {
		fmt.Fprintln(w, "ID\tNAME\tHOST\tSTATUS\tTHRESHOLD\tLAST_REPORTED\tLABELS")
	}

	for _, job := range jobs {
		labelsStr := formatLabels(job.Labels)
		lastReported := job.LastReportedAt.Format("2006-01-02 15:04:05")

		if showApiKeys {
			maskedApiKey := maskApiKey(job.ApiKey)
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%ds\t%s\t%s\n",
				job.ID, job.Name, job.Host, maskedApiKey, job.Status, job.AutomaticFailureThreshold,
				lastReported, labelsStr)
		} else {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%ds\t%s\t%s\n",
				job.ID, job.Name, job.Host, job.Status, job.AutomaticFailureThreshold,
				lastReported, labelsStr)
		}
	}

	w.Flush()
}

// printJobDetails prints detailed job information
func printJobDetails(job *model.Job) {
	fmt.Printf("Job Details:\n")
	fmt.Printf("  ID: %d\n", job.ID)
	fmt.Printf("  Name: %s\n", job.Name)
	fmt.Printf("  Host: %s\n", job.Host)
	fmt.Printf("  API Key: %s\n", job.ApiKey)
	fmt.Printf("  Status: %s\n", job.Status)
	fmt.Printf("  Threshold: %d seconds\n", job.AutomaticFailureThreshold)
	fmt.Printf("  Last Reported: %s\n", job.LastReportedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("  Created: %s\n", job.CreatedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("  Updated: %s\n", job.UpdatedAt.Format("2006-01-02 15:04:05 MST"))

	if len(job.Labels) > 0 {
		fmt.Printf("  Labels:\n")
		for key, value := range job.Labels {
			fmt.Printf("    %s: %s\n", key, value)
		}
	} else {
		fmt.Printf("  Labels: none\n")
	}
}

// formatLabels formats labels map for display
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "-"
	}

	var parts []string
	for key, value := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, ",")
}

// maskApiKey masks an API key for display, showing only the first and last few characters
func maskApiKey(apiKey string) string {
	if len(apiKey) <= 10 {
		return "***"
	}
	return apiKey[:6] + "..." + apiKey[len(apiKey)-4:]
}

// parseJobID parses a job ID from a string argument
func parseJobID(idStr string) (int, error) {
	if idStr == "" {
		return 0, fmt.Errorf("job ID cannot be empty")
	}

	jobID := 0
	if _, err := fmt.Sscanf(idStr, "%d", &jobID); err != nil {
		return 0, fmt.Errorf("job ID must be a number: %s", idStr)
	}

	if jobID <= 0 {
		return 0, fmt.Errorf("job ID must be a positive number: %d", jobID)
	}

	return jobID, nil
}

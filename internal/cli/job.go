package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jaep/cron-exporter/pkg/model"
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
	jobName      string
	jobHost      string
	jobThreshold int
	jobLabels    []string
	jobStatus    string
)

func init() {
	jobAddCmd.Flags().StringVarP(&jobName, "name", "n", "", "job name (required)")
	jobAddCmd.Flags().StringVar(&jobHost, "host", "", "host name (required)")
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
		AutomaticFailureThreshold: jobThreshold,
		Labels:                    labels,
		Status:                    jobStatus,
		LastReportedAt:            time.Now().UTC(),
	}

	if err := jobStore.CreateJob(job); err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	fmt.Printf("Job '%s@%s' created successfully\n", jobName, jobHost)
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
	listLabels []string
	outputJSON bool
)

func init() {
	jobListCmd.Flags().StringSliceVarP(&listLabels, "label", "l", []string{}, "filter by labels in key=value format")
	jobListCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "output as JSON")
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
	Use:   "update",
	Short: "Update a job",
	Long:  `Update an existing job's configuration`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runJobUpdate(cmd); err != nil {
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
	jobUpdateCmd.Flags().StringVarP(&jobName, "name", "n", "", "job name (required)")
	jobUpdateCmd.Flags().StringVar(&jobHost, "host", "", "host name (required)")
	jobUpdateCmd.Flags().IntVar(&jobThreshold, "threshold", 0, "automatic failure threshold in seconds")
	jobUpdateCmd.Flags().StringSliceVarP(&updateLabels, "label", "l", []string{}, "labels in key=value format")
	jobUpdateCmd.Flags().StringVarP(&updateStatus, "status", "s", "", "job status (active, maintenance, paused)")
	jobUpdateCmd.Flags().BoolVarP(&maintenance, "maintenance", "m", false, "set job to maintenance mode")

	jobUpdateCmd.MarkFlagRequired("name")
	jobUpdateCmd.MarkFlagRequired("host")
}

func runJobUpdate(cmd *cobra.Command) error {
	if jobName == "" || jobHost == "" {
		return fmt.Errorf("job name and host are required")
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
	job, err := jobStore.GetJob(jobName, jobHost)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Update fields if provided
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
	if err := jobStore.UpdateJob(job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	fmt.Printf("Job '%s@%s' updated successfully\n", jobName, jobHost)
	return nil
}

// jobDeleteCmd deletes a job
var jobDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a job",
	Long:  `Delete a job definition`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runJobDelete(cmd); err != nil {
			logrus.WithError(err).Fatal("failed to delete job")
		}
	},
}

func init() {
	jobDeleteCmd.Flags().StringVarP(&jobName, "name", "n", "", "job name (required)")
	jobDeleteCmd.Flags().StringVar(&jobHost, "host", "", "host name (required)")

	jobDeleteCmd.MarkFlagRequired("name")
	jobDeleteCmd.MarkFlagRequired("host")
}

func runJobDelete(cmd *cobra.Command) error {
	if jobName == "" || jobHost == "" {
		return fmt.Errorf("job name and host are required")
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

	// Delete job
	if err := jobStore.DeleteJob(jobName, jobHost); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	fmt.Printf("Job '%s@%s' deleted successfully\n", jobName, jobHost)
	return nil
}

// jobShowCmd shows detailed job information
var jobShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show job details",
	Long:  `Show detailed information about a specific job`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runJobShow(cmd); err != nil {
			logrus.WithError(err).Fatal("failed to show job")
		}
	},
}

func init() {
	jobShowCmd.Flags().StringVarP(&jobName, "name", "n", "", "job name (required)")
	jobShowCmd.Flags().StringVar(&jobHost, "host", "", "host name (required)")
	jobShowCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "output as JSON")

	jobShowCmd.MarkFlagRequired("name")
	jobShowCmd.MarkFlagRequired("host")
}

func runJobShow(cmd *cobra.Command) error {
	if jobName == "" || jobHost == "" {
		return fmt.Errorf("job name and host are required")
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

	// Get job
	job, err := jobStore.GetJob(jobName, jobHost)
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
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tHOST\tSTATUS\tTHRESHOLD\tLAST_REPORTED\tLABELS")

	for _, job := range jobs {
		labelsStr := formatLabels(job.Labels)
		lastReported := job.LastReportedAt.Format("2006-01-02 15:04:05")

		fmt.Fprintf(w, "%s\t%s\t%s\t%ds\t%s\t%s\n",
			job.Name, job.Host, job.Status, job.AutomaticFailureThreshold,
			lastReported, labelsStr)
	}

	w.Flush()
}

// printJobDetails prints detailed job information
func printJobDetails(job *model.Job) {
	fmt.Printf("Job Details:\n")
	fmt.Printf("  Name: %s\n", job.Name)
	fmt.Printf("  Host: %s\n", job.Host)
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

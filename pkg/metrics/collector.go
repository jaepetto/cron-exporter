package metrics

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jaep/cron-exporter/pkg/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
)

// Collector implements Prometheus metrics collection for cron jobs
type Collector struct {
	jobStore       *model.JobStore
	jobResultStore *model.JobResultStore
	registry       *prometheus.Registry

	// Metrics
	jobStatus       *prometheus.GaugeVec
	jobStatusReason *prometheus.GaugeVec
	jobLastRun      *prometheus.GaugeVec
	jobDuration     *prometheus.GaugeVec
	totalJobs       prometheus.Gauge
}

// NewCollector creates a new metrics collector
func NewCollector(jobStore *model.JobStore, jobResultStore *model.JobResultStore) *Collector {
	collector := &Collector{
		jobStore:       jobStore,
		jobResultStore: jobResultStore,
		registry:       prometheus.NewRegistry(),
	}

	// Define metrics - use only fixed labels, dynamic labels will be added at runtime
	collector.jobStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_status",
			Help: "Status of cron job: 1=success, 0=failure, -1=maintenance/paused",
		},
		[]string{"job_name", "host"}, // Start with base labels only
	)

	collector.jobStatusReason = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_status_reason",
			Help: "Reason for current job status",
		},
		[]string{"job_name", "host", "reason"},
	)

	collector.jobLastRun = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_last_run_timestamp",
			Help: "Timestamp of last job execution",
		},
		[]string{"job_name", "host"},
	)

	collector.jobDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_duration_seconds",
			Help: "Duration of last job execution in seconds",
		},
		[]string{"job_name", "host"},
	)

	collector.totalJobs = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cronjob_total",
			Help: "Total number of registered cron jobs",
		},
	)

	return collector
}

// Register registers the collector with Prometheus
func (c *Collector) Register() error {
	// Register metrics with our custom registry
	if err := c.registry.Register(c.jobStatus); err != nil {
		return fmt.Errorf("failed to register job_status metric: %w", err)
	}

	if err := c.registry.Register(c.jobStatusReason); err != nil {
		return fmt.Errorf("failed to register job_status_reason metric: %w", err)
	}

	if err := c.registry.Register(c.jobLastRun); err != nil {
		return fmt.Errorf("failed to register job_last_run metric: %w", err)
	}

	if err := c.registry.Register(c.jobDuration); err != nil {
		return fmt.Errorf("failed to register job_duration metric: %w", err)
	}

	if err := c.registry.Register(c.totalJobs); err != nil {
		return fmt.Errorf("failed to register total_jobs metric: %w", err)
	}

	logrus.Info("prometheus metrics registered successfully")
	return nil
}

// Gather collects and returns metrics in Prometheus format
func (c *Collector) Gather() (string, error) {
	// Get all jobs and generate manual metrics
	jobs, err := c.jobStore.ListJobs(nil)
	if err != nil {
		return "", fmt.Errorf("failed to list jobs: %w", err)
	}

	var builder strings.Builder
	now := time.Now().UTC()

	// Write help and type comments
	builder.WriteString("# HELP cronjob_status Status of cron job: 1=success, 0=failure, -1=maintenance/paused\n")
	builder.WriteString("# TYPE cronjob_status gauge\n")

	// Generate job status metrics
	for _, job := range jobs {
		status, reason := c.calculateJobStatus(job, now)

		// Build labels string
		var labels []string
		labels = append(labels, fmt.Sprintf(`job_name="%s"`, job.Name))
		labels = append(labels, fmt.Sprintf(`host="%s"`, job.Host))

		// Add user-defined labels
		for k, v := range job.Labels {
			labels = append(labels, fmt.Sprintf(`%s="%s"`, k, v))
		}

		// Always add status label based on the calculated reason
		if reason != "" {
			labels = append(labels, fmt.Sprintf(`status="%s"`, reason))
		}

		labelsStr := strings.Join(labels, ",")
		builder.WriteString(fmt.Sprintf("cronjob_status{%s} %g\n", labelsStr, status))
	}

	// Write last run timestamps
	builder.WriteString("# HELP cronjob_last_run_timestamp Timestamp of last job execution\n")
	builder.WriteString("# TYPE cronjob_last_run_timestamp gauge\n")
	for _, job := range jobs {
		builder.WriteString(fmt.Sprintf("cronjob_last_run_timestamp{job_name=\"%s\",host=\"%s\"} %d\n",
			job.Name, job.Host, job.LastReportedAt.Unix()))
	}

	// Write total jobs
	builder.WriteString("# HELP cronjob_total Total number of registered cron jobs\n")
	builder.WriteString("# TYPE cronjob_total gauge\n")
	builder.WriteString(fmt.Sprintf("cronjob_total %d\n", len(jobs)))

	return builder.String(), nil
}

// Handler returns an HTTP handler for Prometheus metrics scraping
func (c *Collector) Handler() http.Handler {
	return promhttp.HandlerFor(c.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// updateMetrics updates all metrics with current job data
func (c *Collector) updateMetrics() error {
	// Clear existing metrics
	c.jobStatus.Reset()
	c.jobStatusReason.Reset()
	c.jobLastRun.Reset()
	c.jobDuration.Reset()

	// Get all jobs
	jobs, err := c.jobStore.ListJobs(nil)
	if err != nil {
		return fmt.Errorf("failed to list jobs: %w", err)
	}

	c.totalJobs.Set(float64(len(jobs)))

	now := time.Now().UTC()

	for _, job := range jobs {
		// Create base labels from job metadata
		statusLabels := prometheus.Labels{
			"job_name": job.Name,
			"host":     job.Host,
		}

		// Add user-defined labels to status labels
		for k, v := range job.Labels {
			statusLabels[k] = v
		}

		// Determine job status and reason
		status, reason := c.calculateJobStatus(job, now)

		// Set status metric with all labels
		c.jobStatus.With(statusLabels).Set(status)

		// Set reason metric if there's a specific reason
		if reason != "" {
			reasonLabels := prometheus.Labels{
				"job_name": job.Name,
				"host":     job.Host,
				"reason":   reason,
			}
			c.jobStatusReason.With(reasonLabels).Set(1)
		}

		// Set last run timestamp
		lastRunLabels := prometheus.Labels{
			"job_name": job.Name,
			"host":     job.Host,
		}
		c.jobLastRun.With(lastRunLabels).Set(float64(job.LastReportedAt.Unix()))

		// TODO: Set duration from last job result
		// This would require querying job results, which we'll implement later
	}

	return nil
}

// calculateJobStatus determines the current status and reason for a job
func (c *Collector) calculateJobStatus(job *model.Job, now time.Time) (float64, string) {
	// Jobs in maintenance or paused status
	if job.Status == "maintenance" {
		return -1, "maintenance"
	}
	if job.Status == "paused" {
		return -1, "paused"
	}

	// Check if job has exceeded its failure threshold
	timeSinceLastReport := now.Sub(job.LastReportedAt)
	thresholdDuration := time.Duration(job.AutomaticFailureThreshold) * time.Second

	if timeSinceLastReport > thresholdDuration {
		return 0, "missed_deadline"
	}

	// Get the most recent job result to determine actual status
	if c.jobResultStore != nil {
		results, err := c.jobResultStore.GetJobResults(job.Name, job.Host, 1)
		if err == nil && len(results) > 0 {
			lastResult := results[0]
			if lastResult.Status == "success" {
				return 1, "success"
			} else if lastResult.Status == "failure" {
				return 0, "failure"
			}
		}
	}

	// Fallback: assume success if within threshold and not in maintenance
	return 1, "success"
}

// writeMetricFamily writes a metric family in Prometheus text format
func (c *Collector) writeMetricFamily(builder *strings.Builder, mf *dto.MetricFamily) error {
	metricName := mf.GetName()
	metricType := mf.GetType()

	// Write HELP comment
	if help := mf.GetHelp(); help != "" {
		builder.WriteString(fmt.Sprintf("# HELP %s %s\n", metricName, help))
	}

	// Write TYPE comment
	builder.WriteString(fmt.Sprintf("# TYPE %s %s\n", metricName, strings.ToLower(metricType.String())))

	// Write metrics
	for _, metric := range mf.GetMetric() {
		builder.WriteString(metricName)

		// Write labels
		if len(metric.GetLabel()) > 0 {
			builder.WriteString("{")
			first := true
			for _, label := range metric.GetLabel() {
				if !first {
					builder.WriteString(",")
				}
				builder.WriteString(fmt.Sprintf(`%s="%s"`, label.GetName(), label.GetValue()))
				first = false
			}
			builder.WriteString("}")
		}

		// Write value
		var value float64
		switch metricType {
		case dto.MetricType_COUNTER:
			value = metric.GetCounter().GetValue()
		case dto.MetricType_GAUGE:
			value = metric.GetGauge().GetValue()
		case dto.MetricType_HISTOGRAM:
			// Handle histogram if needed
			continue
		case dto.MetricType_SUMMARY:
			// Handle summary if needed
			continue
		default:
			continue
		}

		builder.WriteString(fmt.Sprintf(" %g", value))

		// Write timestamp if present
		if metric.GetTimestampMs() != 0 {
			builder.WriteString(fmt.Sprintf(" %d", metric.GetTimestampMs()))
		}

		builder.WriteString("\n")
	}

	return nil
}

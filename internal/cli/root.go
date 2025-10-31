package cli

import (
	"fmt"
	"os"

	"github.com/jaepetto/cron-exporter/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	dev     bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cronmetrics",
	Short: "Cron Metrics Collector & Exporter",
	Long: `A Go-based API and web server to centralize cron job results
and export their statuses as Prometheus-compatible metrics.

Features:
- Central REST API for job result submissions
- Prometheus /metrics endpoint with per-job status and labels
- Per-job automatic failure threshold detection
- Maintenance mode to suppress alerting
- Full CRUD operations for job management`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize logging early
		initLogging()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/cronmetrics/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&dev, "dev", false, "run in development mode with debug logging and in-memory database")

	// Add subcommands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(jobCmd)
	rootCmd.AddCommand(configCmd)
}

// initLogging initializes the logging system
func initLogging() {
	if dev {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
		return
	}

	// Load config to get logging settings
	cfg, err := loadConfig()
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel)
		logrus.SetFormatter(&logrus.JSONFormatter{})
		return
	}

	// Set log level
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// Set log format
	if cfg.Logging.Format == "text" {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	} else {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	// Set log output
	if cfg.Logging.Output != "stdout" && cfg.Logging.Output != "stderr" {
		file, err := os.OpenFile(cfg.Logging.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			logrus.SetOutput(file)
		}
	}
}

// loadConfig loads the configuration with proper precedence
func loadConfig() (*config.Config, error) {
	if dev {
		return config.LoadDev()
	}

	configPath := cfgFile
	if configPath == "" {
		configPath = "/etc/cronmetrics/config.yaml"
	}

	return config.Load(configPath)
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `Generate example configuration and manage config settings`,
}

func init() {
	configCmd.AddCommand(configExampleCmd)
}

// configExampleCmd generates example configuration
var configExampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Generate example configuration file",
	Long:  `Generate an example configuration file with all available options`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(config.GetConfigExample())
	},
}

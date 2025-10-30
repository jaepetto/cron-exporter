package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Security SecurityConfig `mapstructure:"security"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Path            string `mapstructure:"path"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

// MetricsConfig holds Prometheus metrics configuration
type MetricsConfig struct {
	Path string `mapstructure:"path"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"` // "json" or "text"
	Output string `mapstructure:"output"` // "stdout", "stderr", or file path
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	APIKeys      []string `mapstructure:"api_keys"`
	AdminAPIKeys []string `mapstructure:"admin_api_keys"`
	RequireHTTPS bool     `mapstructure:"require_https"`
	TLSCertFile  string   `mapstructure:"tls_cert_file"`
	TLSKeyFile   string   `mapstructure:"tls_key_file"`
}

// Load loads configuration from file and environment variables
func Load(configFile string) (*Config, error) {
	// Set default values
	setDefaults()

	// Set environment variable prefix
	viper.SetEnvPrefix("CRONMETRICS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read from config file if provided
	if configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
		}
	}

	// Unmarshal configuration
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// LoadDev loads development configuration with sensible defaults
func LoadDev() (*Config, error) {
	setDefaults()

	// Override with development-specific settings
	viper.Set("database.path", "/tmp/cronmetrics_dev.db")
	viper.Set("logging.level", "debug")
	viper.Set("security.require_https", false)

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dev config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.idle_timeout", 120)

	// Database defaults
	viper.SetDefault("database.path", "/var/lib/cronmetrics/cronmetrics.db")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", 300) // 5 minutes

	// Metrics defaults
	viper.SetDefault("metrics.path", "/metrics")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")

	// Security defaults
	viper.SetDefault("security.require_https", true)
	viper.SetDefault("security.api_keys", []string{})
	viper.SetDefault("security.admin_api_keys", []string{})
}

// validateConfig validates the loaded configuration
func validateConfig(config *Config) error {
	// Validate server configuration
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	// Validate logging level
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true, "panic": true,
	}
	if !validLevels[strings.ToLower(config.Logging.Level)] {
		return fmt.Errorf("invalid logging level: %s", config.Logging.Level)
	}

	// Validate logging format
	if config.Logging.Format != "json" && config.Logging.Format != "text" {
		return fmt.Errorf("invalid logging format: %s (must be 'json' or 'text')", config.Logging.Format)
	}

	// Validate HTTPS configuration
	if config.Security.RequireHTTPS {
		if config.Security.TLSCertFile == "" || config.Security.TLSKeyFile == "" {
			return fmt.Errorf("TLS cert and key files must be specified when HTTPS is required")
		}
	}

	// Validate database path is not empty
	if config.Database.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	return nil
}

// GetConfigExample returns an example configuration file content
func GetConfigExample() string {
	return `# Cron Metrics Collector Configuration

server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 120

database:
  path: "/var/lib/cronmetrics/cronmetrics.db"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 300

metrics:
  path: "/metrics"

logging:
  level: "info"        # debug, info, warn, error, fatal, panic
  format: "json"       # json or text
  output: "stdout"     # stdout, stderr, or file path

security:
  require_https: true
  tls_cert_file: "/etc/ssl/certs/cronmetrics.crt"
  tls_key_file: "/etc/ssl/private/cronmetrics.key"
  api_keys:
    - "your-api-key-here"
  admin_api_keys:
    - "your-admin-api-key-here"

# Environment variable overrides:
# CRONMETRICS_SERVER_PORT=9090
# CRONMETRICS_DATABASE_PATH=/custom/path/db.sqlite
# CRONMETRICS_LOGGING_LEVEL=debug
`
}

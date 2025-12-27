package tlog

import "time"

// Config contains all configuration options for the logger.
type Config struct {
	// Environment determines the log format.
	// "development" uses colored console output, "production" uses JSON format.
	Environment string

	// Level sets the minimum log level.
	// Valid values: "debug", "info", "warn", "error"
	Level string

	// AppName is used as the service identifier in structured logs.
	AppName string

	// Version is the application version for structured logs.
	Version string

	// File output configuration
	EnableFile bool   // Enable file output
	FilePath   string // Path to log file (e.g., "logs/app.log")
	MaxSizeMB  int    // Maximum size in MB before rotation
	MaxBackups int    // Number of old files to keep (0 = unlimited)
	MaxAgeDays int    // Maximum age in days to keep (0 = unlimited)
	Compress   bool   // Compress rotated files

	// Console output
	EnableConsole bool // Enable console (stdout) output

	// Timezone for log timestamps
	Timezone *time.Location
}

// DefaultConfig returns a Config with sensible defaults.
// - Environment: "development"
// - Level: "info"
// - EnableConsole: true
// - EnableFile: false
// - Timezone: Vietnam (UTC+7)
func DefaultConfig() Config {
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	if loc == nil {
		loc = time.FixedZone("UTC+7", 7*60*60)
	}

	return Config{
		Environment:   "development",
		Level:         "info",
		AppName:       "app",
		Version:       "1.0.0",
		EnableConsole: true,
		EnableFile:    false,
		FilePath:      "logs/app.log",
		MaxSizeMB:     100,
		MaxBackups:    3,
		MaxAgeDays:    30,
		Compress:      true,
		Timezone:      loc,
	}
}

// WithEnvironment sets the environment.
func (c Config) WithEnvironment(env string) Config {
	c.Environment = env
	return c
}

// WithLevel sets the log level.
func (c Config) WithLevel(level string) Config {
	c.Level = level
	return c
}

// WithAppName sets the application name.
func (c Config) WithAppName(name string) Config {
	c.AppName = name
	return c
}

// WithVersion sets the application version.
func (c Config) WithVersion(version string) Config {
	c.Version = version
	return c
}

// WithFile enables file output with the given path.
func (c Config) WithFile(path string) Config {
	c.EnableFile = true
	c.FilePath = path
	return c
}

// WithFileRotation configures file rotation settings.
func (c Config) WithFileRotation(maxSizeMB, maxBackups, maxAgeDays int, compress bool) Config {
	c.MaxSizeMB = maxSizeMB
	c.MaxBackups = maxBackups
	c.MaxAgeDays = maxAgeDays
	c.Compress = compress
	return c
}

// WithTimezone sets the timezone for log timestamps.
func (c Config) WithTimezone(loc *time.Location) Config {
	c.Timezone = loc
	return c
}

// WithConsole enables or disables console output.
func (c Config) WithConsole(enabled bool) Config {
	c.EnableConsole = enabled
	return c
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Level == "" {
		c.Level = "info"
	}
	if c.Environment == "" {
		c.Environment = "development"
	}
	if c.EnableFile && c.FilePath == "" {
		c.FilePath = "logs/app.log"
	}
	if c.MaxSizeMB <= 0 {
		c.MaxSizeMB = 100
	}
	if c.Timezone == nil {
		loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
		if loc == nil {
			loc = time.FixedZone("UTC+7", 7*60*60)
		}
		c.Timezone = loc
	}
	return nil
}

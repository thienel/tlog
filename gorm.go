package tlog

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

// GormConfig contains configuration for the GORM logger adapter.
type GormConfig struct {
	// SlowThreshold is the threshold for marking queries as "slow".
	// Default: 200ms
	SlowThreshold time.Duration

	// IgnoreRecordNotFound skips logging ErrRecordNotFound errors.
	// Default: true
	IgnoreRecordNotFound bool

	// LogLevel sets the GORM log level.
	// Default: gormlogger.Warn
	LogLevel gormlogger.LogLevel
}

// DefaultGormConfig returns a GormConfig with sensible defaults.
func DefaultGormConfig() GormConfig {
	return GormConfig{
		SlowThreshold:        200 * time.Millisecond,
		IgnoreRecordNotFound: true,
		LogLevel:             gormlogger.Warn,
	}
}

// GormOption is a function that configures GormConfig.
type GormOption func(*GormConfig)

// WithSlowThreshold sets the slow query threshold.
func WithSlowThreshold(d time.Duration) GormOption {
	return func(c *GormConfig) {
		c.SlowThreshold = d
	}
}

// WithIgnoreRecordNotFound sets whether to ignore ErrRecordNotFound.
func WithIgnoreRecordNotFound(ignore bool) GormOption {
	return func(c *GormConfig) {
		c.IgnoreRecordNotFound = ignore
	}
}

// WithGormLogLevel sets the GORM log level.
func WithGormLogLevel(level gormlogger.LogLevel) GormOption {
	return func(c *GormConfig) {
		c.LogLevel = level
	}
}

// GormLogger is a custom GORM logger that uses tlog.
type GormLogger struct {
	cfg GormConfig
}

// NewGormLogger creates a new GORM logger adapter.
func NewGormLogger(opts ...GormOption) *GormLogger {
	cfg := DefaultGormConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &GormLogger{cfg: cfg}
}

// LogMode sets the log level and returns a new logger.
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.cfg.LogLevel = level
	return &newLogger
}

// Info logs informational messages.
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.cfg.LogLevel >= gormlogger.Info {
		FromContext(ctx).Sugar().Infof(msg, data...)
	}
}

// Warn logs warning messages.
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.cfg.LogLevel >= gormlogger.Warn {
		FromContext(ctx).Sugar().Warnf(msg, data...)
	}
}

// Error logs error messages.
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.cfg.LogLevel >= gormlogger.Error {
		FromContext(ctx).Sugar().Errorf(msg, data...)
	}
}

// parseSQL extracts operation and table name from SQL query.
func parseSQL(sql string) (operation string, table string) {
	sql = strings.TrimSpace(sql)
	upperSQL := strings.ToUpper(sql)

	// Determine operation
	switch {
	case strings.HasPrefix(upperSQL, "SELECT"):
		operation = "SELECT"
	case strings.HasPrefix(upperSQL, "INSERT"):
		operation = "INSERT"
	case strings.HasPrefix(upperSQL, "UPDATE"):
		operation = "UPDATE"
	case strings.HasPrefix(upperSQL, "DELETE"):
		operation = "DELETE"
	default:
		operation = "OTHER"
	}

	// Extract table name based on operation
	table = extractTableName(sql, operation)
	return
}

// extractTableName extracts the table name from SQL based on operation type.
func extractTableName(sql string, operation string) string {
	var pattern *regexp.Regexp

	switch operation {
	case "SELECT":
		// SELECT ... FROM "table_name" or SELECT ... FROM table_name
		pattern = regexp.MustCompile(`(?i)\bFROM\s+["` + "`" + `]?(\w+)["` + "`" + `]?`)
	case "INSERT":
		// INSERT INTO "table_name" or INSERT INTO table_name
		pattern = regexp.MustCompile(`(?i)\bINTO\s+["` + "`" + `]?(\w+)["` + "`" + `]?`)
	case "UPDATE":
		// UPDATE "table_name" or UPDATE table_name
		pattern = regexp.MustCompile(`(?i)\bUPDATE\s+["` + "`" + `]?(\w+)["` + "`" + `]?`)
	case "DELETE":
		// DELETE FROM "table_name" or DELETE FROM table_name
		pattern = regexp.MustCompile(`(?i)\bFROM\s+["` + "`" + `]?(\w+)["` + "`" + `]?`)
	default:
		return ""
	}

	if pattern != nil {
		matches := pattern.FindStringSubmatch(sql)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// Trace logs SQL queries with timing information.
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.cfg.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// Parse SQL to extract operation and table
	operation, table := parseSQL(sql)

	// Check if slow query
	isSlowQuery := elapsed > l.cfg.SlowThreshold

	fields := []zap.Field{
		zap.String("operation", operation),
		zap.String("table", table),
		zap.Int64("duration_ms", elapsed.Milliseconds()),
		zap.Int64("rows_affected", rows),
		zap.String("sql", sql),
		zap.String("caller", utils.FileWithLineNum()),
	}

	// Add slow_query flag if applicable
	if isSlowQuery {
		fields = append(fields, zap.Bool("slow_query", true))
	}

	logger := FromContext(ctx)

	switch {
	// Case 1: Log errors (except record not found if configured to ignore)
	case err != nil && l.cfg.LogLevel >= gormlogger.Error:
		if l.cfg.IgnoreRecordNotFound && errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		fields = append(fields, zap.Error(err))
		logger.Error("Database query failed", fields...)

	// Case 2: Log slow queries
	case isSlowQuery && l.cfg.LogLevel >= gormlogger.Warn:
		logger.Warn("Slow database query detected", fields...)

	// Case 3: Log all queries (Info level)
	case l.cfg.LogLevel >= gormlogger.Info:
		logger.Info("Database query executed", fields...)
	}
}

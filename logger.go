package tlog

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var globalLogger *zap.Logger

// Init initializes the global logger with the provided configuration.
func Init(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Parse log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// Create encoder config based on environment
	encoderConfig := createEncoderConfig(cfg)

	var cores []zapcore.Core

	// Console core
	if cfg.EnableConsole {
		consoleEncoder := createConsoleEncoder(cfg, encoderConfig)
		consoleCore := zapcore.NewCore(
			consoleEncoder,
			zapcore.Lock(os.Stdout),
			level,
		)
		cores = append(cores, consoleCore)
	}

	// File core
	if cfg.EnableFile {
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
		}
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(fileWriter),
			level,
		)
		cores = append(cores, fileCore)
	}

	// If no cores configured, default to console
	if len(cores) == 0 {
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.Lock(os.Stdout),
			level,
		)
		cores = append(cores, consoleCore)
	}

	// Create tee core
	core := zapcore.NewTee(cores...)

	// Build logger with options
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCallerSkip(1),
	)

	// Add global fields
	var globalFields []zap.Field
	if cfg.AppName != "" {
		globalFields = append(globalFields, zap.String("service", cfg.AppName))
	}
	if cfg.Version != "" {
		globalFields = append(globalFields, zap.String("version", cfg.Version))
	}
	if len(globalFields) > 0 {
		logger = logger.With(globalFields...)
	}

	globalLogger = logger
	zap.ReplaceGlobals(logger)

	return nil
}

// InitWithDefaults initializes the logger with default configuration.
func InitWithDefaults() error {
	return Init(DefaultConfig())
}

// createEncoderConfig creates the zapcore.EncoderConfig based on configuration.
func createEncoderConfig(cfg Config) zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     timezoneEncoder(cfg.Timezone),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// createConsoleEncoder creates the appropriate encoder for console output.
func createConsoleEncoder(cfg Config, encoderConfig zapcore.EncoderConfig) zapcore.Encoder {
	if cfg.Environment == "production" {
		return zapcore.NewJSONEncoder(encoderConfig)
	}

	// Development mode: use colored console encoder
	devConfig := zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		CallerKey:      "C",
		MessageKey:     "M",
		StacktraceKey:  "S",
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("15:04:05"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	return zapcore.NewConsoleEncoder(devConfig)
}

// timezoneEncoder creates a time encoder for the specified timezone.
func timezoneEncoder(loc *time.Location) zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.In(loc).Format("2006-01-02T15:04:05.000Z07:00"))
	}
}

// L returns the global logger.
func L() *zap.Logger {
	if globalLogger == nil {
		return zap.L()
	}
	return globalLogger
}

// S returns the global sugared logger.
func S() *zap.SugaredLogger {
	return L().Sugar()
}

// Info logs an info message.
func Info(msg string, fields ...zap.Field) {
	L().Info(msg, fields...)
}

// Error logs an error message.
func Error(msg string, fields ...zap.Field) {
	L().Error(msg, fields...)
}

// Debug logs a debug message.
func Debug(msg string, fields ...zap.Field) {
	L().Debug(msg, fields...)
}

// Warn logs a warning message.
func Warn(msg string, fields ...zap.Field) {
	L().Warn(msg, fields...)
}

// Fatal logs a fatal message and exits.
func Fatal(msg string, fields ...zap.Field) {
	L().Fatal(msg, fields...)
}

// Panic logs a panic message and panics.
func Panic(msg string, fields ...zap.Field) {
	L().Panic(msg, fields...)
}

// With creates a child logger with the given fields.
func With(fields ...zap.Field) *zap.Logger {
	return L().With(fields...)
}

// Sync flushes any buffered log entries.
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

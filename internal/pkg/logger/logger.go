package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	// LogLevelDebug represents debug log level
	LogLevelDebug = "debug"
	// LogLevelInfo represents info log level
	LogLevelInfo = "info"
	// LogLevelWarn represents warn log level
	LogLevelWarn = "warn"
	// LogLevelError represents error log level
	LogLevelError = "error"
	// LogLevelFatal represents fatal log level
	LogLevelFatal = "fatal"
)

// Config represents logger configuration
type Config struct {
	Level       string // debug, info, warn, error, fatal
	Environment string // development, production, test
	LogFile     string // optional file path for logs
}

// Init initializes the global logger with the given configuration
func Init(cfg Config) error {
	// Configure time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Parse log level
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Set up output writers
	var writers []io.Writer
	writers = append(writers, os.Stdout)

	// Add file output if specified
	if cfg.LogFile != "" {
		file, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Error().Err(err).Str("file", cfg.LogFile).Msg("Failed to open log file")
		} else {
			writers = append(writers, file)
		}
	}

	// Create multi-writer
	multiWriter := zerolog.MultiLevelWriter(writers...)

	// Configure output format based on environment
	if cfg.Environment == "development" || cfg.Environment == "dev" {
		// Pretty console output for development
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
			NoColor:    false,
		}).With().Caller().Logger()
	} else {
		// JSON output for production for better parsing
		log.Logger = zerolog.New(multiWriter).
			With().
			Timestamp().
			Caller().
			Logger()
	}

	return nil
}

// FromContext returns the logger from context or the global logger
func FromContext(ctx context.Context) *zerolog.Logger {
	if ctxLogger := ctx.Value(ContextKey); ctxLogger != nil {
		if logger, ok := ctxLogger.(*zerolog.Logger); ok {
			return logger
		}
	}
	return &log.Logger
}

// WithContext returns a context with the logger attached
func WithContext(ctx context.Context, logger *zerolog.Logger) context.Context {
	return context.WithValue(ctx, ContextKey, logger)
}

// ContextKey is the key used to store logger in context
type contextKey string

const ContextKey contextKey = "logger"

// LogError logs an error with context
func LogError(ctx context.Context, err error, msg string, fields ...interface{}) {
	logger := FromContext(ctx)
	event := logger.Error().Err(err)

	// Add fields in pairs (key, value)
	for i := 0; i < len(fields)-1; i += 2 {
		event.Interface(fields[i].(string), fields[i+1])
	}

	event.Msg(msg)
}

// LogInfo logs an info message with context
func LogInfo(ctx context.Context, msg string, fields ...interface{}) {
	logger := FromContext(ctx)
	event := logger.Info()

	// Add fields in pairs (key, value)
	for i := 0; i < len(fields)-1; i += 2 {
		event.Interface(fields[i].(string), fields[i+1])
	}

	event.Msg(msg)
}

// LogWarn logs a warning message with context
func LogWarn(ctx context.Context, msg string, fields ...interface{}) {
	logger := FromContext(ctx)
	event := logger.Warn()

	// Add fields in pairs (key, value)
	for i := 0; i < len(fields)-1; i += 2 {
		event.Interface(fields[i].(string), fields[i+1])
	}

	event.Msg(msg)
}

// LogDebug logs a debug message with context
func LogDebug(ctx context.Context, msg string, fields ...interface{}) {
	logger := FromContext(ctx)
	event := logger.Debug()

	// Add fields in pairs (key, value)
	for i := 0; i < len(fields)-1; i += 2 {
		event.Interface(fields[i].(string), fields[i+1])
	}

	event.Msg(msg)
}

package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log zerolog.Logger

// Config holds logger configuration
type Config struct {
	Level      string
	LogDir     string
	MaxSize    int  // Max size in MB before rotation
	MaxBackups int  // Max number of old log files to retain
	MaxAge     int  // Max number of days to retain old log files
	Compress   bool // Compress rotated files
	Console    bool // Also output to console
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		Level:      getEnv("LOG_LEVEL", "info"),
		LogDir:     getEnv("LOG_DIR", "./logs"),
		MaxSize:    100, // 100 MB
		MaxBackups: 3,
		MaxAge:     28, // 28 days
		Compress:   true,
		Console:    getEnv("ENV", "development") == "development",
	}
}

// Init initializes the global logger with the given configuration
func Init(cfg Config) {
	// Set log level
	level := parseLogLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		panic("failed to create log directory: " + err.Error())
	}

	// Setup file rotation
	fileWriter := &lumberjack.Logger{
		Filename:   filepath.Join(cfg.LogDir, "app.log"),
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	// Setup writers
	var writers []io.Writer
	writers = append(writers, fileWriter)

	// Add console output if enabled
	if cfg.Console {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		writers = append(writers, consoleWriter)
	}

	// Create multi-writer
	multi := io.MultiWriter(writers...)

	// Initialize logger with timestamp and caller info
	Log = zerolog.New(multi).
		With().
		Timestamp().
		Caller().
		Logger()
}

// parseLogLevel converts string log level to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// getEnv gets environment variable or returns default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Info returns a new info level event
func Info() *zerolog.Event {
	return Log.Info()
}

// Error returns a new error level event
func Error() *zerolog.Event {
	return Log.Error()
}

// Debug returns a new debug level event
func Debug() *zerolog.Event {
	return Log.Debug()
}

// Warn returns a new warn level event
func Warn() *zerolog.Event {
	return Log.Warn()
}

// Fatal returns a new fatal level event
func Fatal() *zerolog.Event {
	return Log.Fatal()
}

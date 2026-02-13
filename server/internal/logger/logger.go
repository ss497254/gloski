package logger

import (
	"fmt"
	"log/slog"
	"os"
)

// ParseLevel converts a string log level to slog.Level.
func ParseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

var level = new(slog.LevelVar)

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))
}

// SetLevel sets the global log level.
func SetLevel(l slog.Level) {
	level.Set(l)
}

// Debug logs a message at debug level.
func Debug(format string, args ...interface{}) {
	slog.Debug(fmt.Sprintf(format, args...))
}

// Info logs a message at info level.
func Info(format string, args ...interface{}) {
	slog.Info(fmt.Sprintf(format, args...))
}

// Warn logs a message at warn level.
func Warn(format string, args ...interface{}) {
	slog.Warn(fmt.Sprintf(format, args...))
}

// Error logs a message at error level.
func Error(format string, args ...interface{}) {
	slog.Error(fmt.Sprintf(format, args...))
}

// Fatal logs a message at error level and exits.
func Fatal(format string, args ...interface{}) {
	slog.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

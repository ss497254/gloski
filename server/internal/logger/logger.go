package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

type Logger struct {
	level  Level
	output io.Writer
	mu     sync.Mutex
}

var defaultLogger = &Logger{
	level:  LevelInfo,
	output: os.Stdout,
}

func SetLevel(level Level) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.level = level
}

func SetOutput(w io.Writer) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.output = w
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.output, "%s [%s] %s\n", timestamp, level.String(), msg)
}

func Debug(format string, args ...interface{}) {
	defaultLogger.log(LevelDebug, format, args...)
}

func Info(format string, args ...interface{}) {
	defaultLogger.log(LevelInfo, format, args...)
}

func Warn(format string, args ...interface{}) {
	defaultLogger.log(LevelWarn, format, args...)
}

func Error(format string, args ...interface{}) {
	defaultLogger.log(LevelError, format, args...)
}

func Fatal(format string, args ...interface{}) {
	defaultLogger.log(LevelError, format, args...)
	os.Exit(1)
}

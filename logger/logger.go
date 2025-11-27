package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Level represents log levels
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a string level into Level type
func ParseLevel(levelStr string) Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

// Logger provides logging functionality
type Logger struct {
	level    Level
	logger   *log.Logger
	logFile  *os.File
}

// NewLogger creates a new Logger instance
func NewLogger(levelStr string) (*Logger, error) {
	level := ParseLevel(levelStr)
	
	// Create a logger that outputs to both stdout and optionally to a file
	flags := log.LstdFlags | log.Lshortfile
	
	// For now, we'll use stdout only. In production, you might want to also log to a file
	l := &Logger{
		level:  level,
		logger: log.New(os.Stdout, "", flags),
	}

	return l, nil
}

// NewFileLogger creates a new Logger that also writes to a file
func NewFileLogger(levelStr string, logFilePath string) (*Logger, error) {
	level := ParseLevel(levelStr)
	
	// Open log file
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create a multi writer that writes to both stdout and file
	multiWriter := os.Stdout
	if logFile != nil {
		multiWriter = logFile
	}

	flags := log.LstdFlags | log.Lshortfile
	l := &Logger{
		level:   level,
		logger:  log.New(multiWriter, "", flags),
		logFile: logFile,
	}

	return l, nil
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.level <= DEBUG {
		l.log("DEBUG", msg, args...)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	if l.level <= INFO {
		l.log("INFO", msg, args...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	if l.level <= WARN {
		l.log("WARN", msg, args...)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	if l.level <= ERROR {
		l.log("ERROR", msg, args...)
	}
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log("FATAL", msg, args...)
	os.Exit(1)
}

// log handles the actual logging
func (l *Logger) log(level, msg string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formattedMsg := fmt.Sprintf(msg, args...)
	l.logger.Printf("[%s] [%s] %s", timestamp, level, formattedMsg)
}

// Close closes the logger and any open files
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}

// WithFields returns a new logger with additional fields (placeholder for structured logging)
func (l *Logger) WithFields(fields map[string]interface{}) Logger {
	// For simplicity, we'll just append the fields to the message
	// In a real implementation, you might want to use structured logging
	return *l
}
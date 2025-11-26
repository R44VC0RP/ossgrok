package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Level represents the log level
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel = INFO
	logger       = log.New(os.Stdout, "", log.LstdFlags)
)

// SetLevel sets the current log level
func SetLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		currentLevel = DEBUG
	case "info":
		currentLevel = INFO
	case "warn":
		currentLevel = WARN
	case "error":
		currentLevel = ERROR
	default:
		currentLevel = INFO
	}
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	if currentLevel <= DEBUG {
		logger.Printf("[DEBUG] "+format, args...)
	}
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	if currentLevel <= INFO {
		logger.Printf("[INFO] "+format, args...)
	}
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	if currentLevel <= WARN {
		logger.Printf("[WARN] "+format, args...)
	}
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	if currentLevel <= ERROR {
		logger.Printf("[ERROR] "+format, args...)
	}
}

// Fatal logs a fatal error and exits
func Fatal(format string, args ...interface{}) {
	logger.Printf("[FATAL] "+format, args...)
	os.Exit(1)
}

// Printf provides a simple printf-style logging
func Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

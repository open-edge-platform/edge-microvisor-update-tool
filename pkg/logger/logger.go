package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

var (
	timestampEnabled = true
	debugEnabled     = os.Getenv("DEBUG") == "true"
)

// getTimestamp returns the current timestamp if enabled.
func getTimestamp() string {
	if timestampEnabled {
		return time.Now().Format(time.RFC3339)
	}
	return ""
}

// LogError logs an error message with a timestamp.
func LogError(format string, args ...interface{}) {
	timestamp := getTimestamp()
	log.Printf("%s \033[31m[ERROR]\033[0m: %s\n", timestamp, fmt.Sprintf(format, args...))
}

// LogWarn logs a warning message with a timestamp.
func LogWarn(format string, args ...interface{}) {
	timestamp := getTimestamp()
	log.Printf("%s \033[93m[WARN]\033[0m: %s\n", timestamp, fmt.Sprintf(format, args...))
}

// LogInfo logs an informational message with a timestamp.
func LogInfo(format string, args ...interface{}) {
	timestamp := getTimestamp()
	log.Printf("%s [INFO]: %s\n", timestamp, fmt.Sprintf(format, args...))
}

// LogDebug logs a debug message with a timestamp if debugging is enabled.
func LogDebug(format string, args ...interface{}) {
	if debugEnabled {
		timestamp := getTimestamp()
		log.Printf("%s [DEBUG]: %s\n", timestamp, fmt.Sprintf(format, args...))
	}
}

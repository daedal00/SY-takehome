package platform

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Logger provides structured logging with key-value pairs
type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
}

// NewLogger creates a new Logger instance
func NewLogger() *Logger {
	return &Logger{
		infoLogger:  log.New(os.Stdout, "INFO: ", log.LstdFlags),
		errorLogger: log.New(os.Stderr, "ERROR: ", log.LstdFlags),
	}
}

// Info logs an informational message with structured key-value pairs
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.infoLogger.Println(formatMessage(msg, keysAndValues...))
}

// Error logs an error message with structured key-value pairs
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.errorLogger.Println(formatMessage(msg, keysAndValues...))
}

// formatMessage formats a message with key-value pairs
func formatMessage(msg string, keysAndValues ...interface{}) string {
	var sb strings.Builder
	sb.WriteString(msg)
	
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			sb.WriteString(fmt.Sprintf("\n  %v=%v", keysAndValues[i], keysAndValues[i+1]))
		}
	}
	
	return sb.String()
}

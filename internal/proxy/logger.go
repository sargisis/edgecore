package proxy

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// LogFormat determines the output format for logs.
type LogFormat string

const (
	LogFormatPretty LogFormat = "pretty"
	LogFormatJSON   LogFormat = "json"
)

var (
	// logFormat controls whether logs are JSON or pretty-printed.
	logFormat LogFormat = LogFormatPretty
)

// SetLogFormat sets the log output format (json or pretty).
func SetLogFormat(format LogFormat) {
	logFormat = format
}

// LogEntry represents a structured log entry.
type LogEntry struct {
	Time      string  `json:"time"`
	Level     string  `json:"level"`
	Message   string  `json:"message,omitempty"`
	ClientIP  string  `json:"client_ip,omitempty"`
	Method    string  `json:"method,omitempty"`
	Path      string  `json:"path,omitempty"`
	Status    int     `json:"status,omitempty"`
	Duration  float64 `json:"duration_seconds,omitempty"`
	RequestID string  `json:"request_id,omitempty"`
	Backend   string  `json:"backend,omitempty"`
	Error     string  `json:"error,omitempty"`
}

// logEntry writes a structured log entry.
func logEntry(entry LogEntry) {
	entry.Time = time.Now().Format(time.RFC3339)

	if logFormat == LogFormatJSON {
		data, err := json.Marshal(entry)
		if err != nil {
			log.Printf("failed to marshal log entry: %v", err)
			return
		}
		fmt.Fprintln(os.Stderr, string(data))
	} else {
		// Pretty format for development
		if entry.RequestID != "" {
			log.Printf("[%s] %s %s %s (client: %s, duration: %.3fs, request_id: %s)",
				entry.Level, entry.Method, entry.Path,
				statusText(entry.Status), entry.ClientIP, entry.Duration, entry.RequestID)
		} else if entry.Message != "" {
			log.Printf("[%s] %s", entry.Level, entry.Message)
		} else {
			log.Printf("[%s] %s", entry.Level, entry.Message)
		}
	}
}

func statusText(status int) string {
	if status == 0 {
		return ""
	}
	return fmt.Sprintf("status=%d", status)
}

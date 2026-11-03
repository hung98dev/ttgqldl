package telemetry

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewLoggerWritesJSONAtRequestedLevel(t *testing.T) {
	var buf bytes.Buffer
	logger, err := NewLogger(&buf, "info", "test-service")
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	logger.Debug("hidden")
	logger.Info("visible", "key", "value")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one log line, got %d: %q", len(lines), buf.String())
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("log line is not JSON: %v (%q)", err, lines[0])
	}
	if entry["msg"] != "visible" {
		t.Errorf("msg = %v", entry["msg"])
	}
	if entry["level"] != "INFO" {
		t.Errorf("level = %v", entry["level"])
	}
	if entry["service"] != "test-service" {
		t.Errorf("service = %v", entry["service"])
	}
	if entry["key"] != "value" {
		t.Errorf("attribute key = %v", entry["key"])
	}
}

func TestNewLoggerRejectsUnknownLevel(t *testing.T) {
	var buf bytes.Buffer
	if _, err := NewLogger(&buf, "loud", "svc"); err == nil {
		t.Fatal("expected an error for an unknown level")
	}
}

// Package telemetry builds the structured logger used by the backend.
//
// The backend logs JSON to stdout so that a container runtime, systemd unit or
// log shipper can parse it without a custom decoder. Levels are validated by
// the config package before reaching this package.
package telemetry

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// NewLogger returns a JSON slog.Logger writing to w at the requested level.
func NewLogger(w io.Writer, level, service string, attrs ...slog.Attr) (*slog.Logger, error) {
	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return nil, fmt.Errorf("unsupported log level %q", level)
	}
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl})
	logger := slog.New(handler)
	if service != "" {
		logger = logger.With(slog.String("service", service))
	}
	if len(attrs) > 0 {
		args := make([]any, 0, len(attrs))
		for _, a := range attrs {
			args = append(args, a)
		}
		logger = logger.With(args...)
	}
	return logger, nil
}

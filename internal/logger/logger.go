// Package logger constructs the application's slog.Logger. It takes
// primitives rather than a config struct so it stays trivially testable and
// independent of how settings are loaded.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Supported output formats. Text is for a terminal during development; JSON
// is for a log aggregator in production.
const (
	FormatText = "text"
	FormatJSON = "json"
)

// New builds a logger writing to w. AddSource is on: the file:line of a log
// call is the fastest way to find the code behind a production error, and
// the cost is paid only on lines that actually get emitted.
func New(w io.Writer, format string, level slog.Level) (*slog.Logger, error) {
	opts := slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	var h slog.Handler
	switch strings.ToLower(format) {
	case FormatText:
		h = slog.NewTextHandler(w, &opts)
	case FormatJSON:
		h = slog.NewJSONHandler(w, &opts)
	default:
		return nil, fmt.Errorf("logger: unknown format %q: want %q or %q", format, FormatText, FormatJSON)
	}

	return slog.New(h), nil
}

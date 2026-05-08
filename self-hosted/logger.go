package main

import (
	"log/slog"
	"os"
	"strings"
)

// newLogger builds a JSON slog logger writing to stdout at the given level.
// Unknown levels fall back to info and emit a warning on the new logger.
func newLogger(level string) *slog.Logger {
	lvl := parseLevel(level)
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	l := slog.New(h)
	if !knownLevel(level) {
		l.Warn("unknown LOG_LEVEL, using info", "got", level)
	}
	return l
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func knownLevel(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "info", "debug", "warn", "warning", "error":
		return true
	}
	return false
}

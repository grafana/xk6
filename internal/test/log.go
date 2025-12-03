package test

import (
	"context"
	"log/slog"
	"strings"
)

type logEntry map[string]any

func (le logEntry) level() slog.Level {
	val, ok := le["level"]
	if !ok {
		return slog.LevelInfo
	}

	levelStr, ok := val.(string)
	if !ok {
		return slog.LevelInfo
	}

	return parseLogLevel(levelStr)
}

func (le logEntry) log(ctx context.Context, extra ...any) string {
	var reason string

	level := le.level()

	var msg string

	val, ok := le["msg"]
	if ok {
		msg, _ = val.(string)
	}

	// Capture jslib testing abort reason
	if strings.HasPrefix(msg, "test aborted:") {
		reason = msg
	}

	args := make([]any, 0)

	for k, v := range le {
		if k == "level" || k == "msg" || k == "time" {
			continue
		}

		// Capture error reason if not already set
		if k == "error" && len(reason) == 0 {
			reason, _ = v.(string)
		}

		args = append(args, k, v)
	}

	args = append(args, extra...)

	slog.Log(ctx, level, msg, args...)

	return reason
}

func parseLogLevel(levelStr string) slog.Level {
	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

package main

import (
	"log/slog"
	"os"
)

var globalLogger *slog.Logger
var defaultLogger *slog.Logger

func initializeLogging() *slog.Logger {
	if globalLogger != nil {
		return globalLogger
	}

	globalLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: mapLogLevels("DEBUG")}))

	globalLogger.Info("Logging initialized!")
	return globalLogger
}

func getLogger() *slog.Logger {
	if defaultLogger != nil {
		return defaultLogger
	}

	defaultLogger = globalLogger
	return defaultLogger
}

func mapLogLevels(level string) slog.Leveler {
	switch level {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

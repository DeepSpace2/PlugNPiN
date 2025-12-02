package logging

import (
	"log/slog"
)

var (
	log   = slog.With()
	DEBUG = slog.LevelDebug
	INFO  = slog.LevelInfo
)

func GetLogger() *slog.Logger {
	return log
}

func SetLevel(level slog.Level) {
	slog.SetLogLoggerLevel(level)
	slog.SetDefault(log)
}

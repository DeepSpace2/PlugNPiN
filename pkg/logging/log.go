package logging

import (
	"log/slog"
	"os"
)

var (
	levelVar = new(slog.LevelVar)
	log      = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: levelVar,
	}))
	DEBUG = slog.LevelDebug
	INFO  = slog.LevelInfo
)

func GetLogger() *slog.Logger {
	return log
}

func SetLevel(level slog.Level) {
	levelVar.Set(level)
	slog.SetDefault(log)
}

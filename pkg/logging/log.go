package logging

import (
	"context"
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

func GetLogger(component string) *slog.Logger {
	return log.With("component", component)
}

func SetLevel(level slog.Level) {
	levelVar.Set(level)
	slog.SetDefault(log)
}

type loggerKey struct{}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}

	return log
}

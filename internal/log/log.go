package log

import (
	"context"
	"log/slog"
	"os"
)

type Logger interface {
	Log(level slog.Level, msg string, args ...any)
}

type GLogger struct {
	ctx context.Context
}

func New(ctx context.Context) *GLogger {
	logHandler := slog.NewJSONHandler(os.Stderr, nil)
	slog.SetDefault(slog.New(logHandler))

	return &GLogger{ctx}
}

func (l *GLogger) Log(level slog.Level, msg string, a ...any) {
	slog.Log(l.ctx, level, msg, a...)
}

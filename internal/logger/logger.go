package logger

import (
	"context"
	"log/slog"
)

type Logger struct {
	l   *slog.Logger
	ctx context.Context
}

func (l *Logger) Info(msg string, args ...slog.Attr) {
	l.Log(slog.LevelInfo, msg, args...)
}

func (l *Logger) Warn(msg string, args ...slog.Attr) {
	l.Log(slog.LevelWarn, msg, args...)
}

func (l *Logger) Error(msg string, args ...slog.Attr) {
	l.Log(slog.LevelError, msg, args...)
}

func (l *Logger) Log(level slog.Level, msg string, attrs ...slog.Attr) {
	attrs = append(attrs, fromContext(l.ctx).attrs...)

	l.l.LogAttrs(l.ctx, level, msg, attrs...)
}

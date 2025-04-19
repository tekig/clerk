package logger

import (
	"context"
	"log/slog"
)

type key int

var loggerKey key

type attributes struct {
	attrs []slog.Attr
}

func NewLogger(ctx context.Context) (context.Context, *Logger) {
	prev := fromContext(ctx)

	next := &attributes{
		attrs: make([]slog.Attr, len(prev.attrs)),
	}

	copy(next.attrs, prev.attrs)

	ctx = context.WithValue(ctx, loggerKey, next)

	return ctx, &Logger{
		ctx: ctx,
		l:   slog.Default(),
	}
}

func WithAttrs(ctx context.Context, attrs ...slog.Attr) {
	ctxAttrs := fromContext(ctx)

	ctxAttrs.attrs = append(ctxAttrs.attrs, attrs...)
}

func fromContext(ctx context.Context) *attributes {
	l, ok := ctx.Value(loggerKey).(*attributes)
	if !ok {
		l = &attributes{}
	}

	return l
}

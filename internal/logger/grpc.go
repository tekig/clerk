package logger

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		startTime := time.Now()

		ctx, logger := NewLogger(ctx)

		resp, err := handler(ctx, req)

		duration := time.Since(startTime)

		var attrs = []slog.Attr{
			slog.Duration("duration", duration),
			slog.String("method", info.FullMethod),
		}

		var level = slog.LevelInfo
		if err != nil {
			level = slog.LevelError
			attrs = append(attrs, slog.String("error", err.Error()))
		}

		logger.Log(
			level, "grpc call",
			attrs...,
		)

		return resp, err
	}
}

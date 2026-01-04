package logger

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

func EchoLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			startTime := time.Now()

			ctx, logger := NewLogger(c.Request().Context())

			c.SetRequest(c.Request().WithContext(ctx))

			err = next(c)

			duration := time.Since(startTime)

			var attrs = []slog.Attr{
				slog.Duration("duration", duration),
				slog.String("path", c.Request().RequestURI),
				slog.String("ip", c.RealIP()),
			}

			var level = slog.LevelInfo
			if err != nil {
				level = slog.LevelError
				attrs = append(attrs, slog.String("error", err.Error()))

			}

			logger.Log(
				level, "grpc http",
				attrs...,
			)

			return err
		}
	}
}

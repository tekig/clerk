package rest

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tekig/clerk/internal/logger"
)

func ConcurrencyLimiter(maxConcurrency int) echo.MiddlewareFunc {
	sem := make(chan struct{}, maxConcurrency)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			select {
			case <-c.Request().Context().Done():
				return c.Request().Context().Err()
			case sem <- struct{}{}:
				defer func() {
					<-sem
				}()

				logger.WithAttrs(c.Request().Context(), slog.Duration("concurrency_limiter", time.Since(start)))

				return next(c)
			}
		}
	}
}

package app

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
)

type App interface {
	Run() error
	Shutdown() error
}

func Start(ctx context.Context, app App) error {
	var errs []error

	var run = make(chan error)
	go func() {
		if err := app.Run(); err != nil {
			run <- fmt.Errorf("run: %w", err)
		}
		close(run)
	}()

	select {
	case err, ok := <-run:
		if !ok {
			err = errors.New("run closed without error")
		}
		errs = append(errs, err)
	case <-ctx.Done():
	}

	if err := app.Shutdown(); err != nil {
		errs = append(errs, fmt.Errorf("shutdown: %w", err))
	}

	return errors.Join(errs...)
}

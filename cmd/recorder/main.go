package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tekig/clerk/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	run, err := app.NewRecorder()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err := app.Start(ctx, run); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

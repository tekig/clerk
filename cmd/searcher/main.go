package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/tekig/clerk/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	run, err := app.NewSearcher()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err := app.Start(ctx, run); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

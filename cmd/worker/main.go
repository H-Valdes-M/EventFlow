package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"eventflow/cmd/app"
)

func main() {
	worker, err := app.InitWorker()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := worker.Run(ctx); err != nil {
		_ = worker.Shutdown()
		log.Fatal(err)
	}
	if err := worker.Shutdown(); err != nil {
		log.Fatal(err)
	}
}

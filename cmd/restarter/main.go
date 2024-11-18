package main

import (
	"context"
	"os/signal"
	"syscall"

	logging "github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

func main() {
	r, err := newRestarter()
	if err != nil {
		log.Fatalf("Failed to create restarter: %v", err)
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)

	err = r.start(ctx)
	if err != nil {
		log.Fatalf("Failed to run restarter: %v", err)
	}
}

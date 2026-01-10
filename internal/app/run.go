package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Runner func(ctx context.Context) error

func Run(serviceName string, run Runner) int {
	log.Printf("%s starting", serviceName)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- run(ctx) }()

	select {
	case <-ctx.Done():
		log.Printf("%s shutting down", serviceName)
		//TODO небольшой grace period (на будущее: закрыть коннекты)
		time.Sleep(200 * time.Millisecond)
		return 0
	case err := <-errCh:
		if err != nil {
			log.Printf("%s failed: %v", serviceName, err)
			return 1
		}
		log.Printf("%s stopped", serviceName)
		return 0
	}
}

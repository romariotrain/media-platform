package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

// Run запускает сервис с graceful shutdown по SIGINT/SIGTERM.
// Возвращает exit code: 0 при успехе, 1 при ошибке.
func Run(name string, fn func(ctx context.Context) error) int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Info().Str("service", name).Msg("starting service")

	if err := fn(ctx); err != nil {
		log.Error().Err(err).Str("service", name).Msg("service failed")
		return 1
	}

	log.Info().Str("service", name).Msg("service stopped")
	return 0
}

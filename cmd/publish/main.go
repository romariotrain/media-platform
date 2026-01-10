package main

import (
	"context"
	"os"

	"github.com/romariotrain/media-platform/internal/app"
)

func main() {
	code := app.Run("publish", func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})
	os.Exit(code)
}

package main

import (
	"context"
	"os"

	"github.com/romariotrain/media-platform/internal/cli"
)

func main() {
	code := cli.Run("ingest", func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})
	os.Exit(code)
}

package main

import (
	"os"

	"github.com/romariotrain/media-platform/internal/cli"
)

func main() {
	code := cli.Run("orchestrator", run)
	os.Exit(code)
}

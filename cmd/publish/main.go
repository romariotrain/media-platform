package main

import (
	"os"

	"github.com/romariotrain/media-platform/internal/cli"
)

func main() {
	code := cli.Run("publish", run)
	os.Exit(code)
}

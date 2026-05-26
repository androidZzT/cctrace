package main

import (
	"context"
	"fmt"
	"os"

	"github.com/agentz/cctrace/internal/app"
)

func main() {
	cfg, err := app.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := app.Run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

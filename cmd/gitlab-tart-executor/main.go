package main

import (
	"context"
	"github.com/cirruslabs/gitlab-tart-executor/internal/commands"
	"log"
	"os"
	"os/signal"
	"strconv"
)

func main() {
	// Set up signal interruptible context
	ctx, cancel := context.WithCancel(context.Background())

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)

	go func() {
		select {
		case <-interruptCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	failureExitCodeRaw := os.Getenv("BUILD_FAILURE_EXIT_CODE")
	if failureExitCodeRaw == "" {
		failureExitCodeRaw = "1" // default
	}
	failureExitCode, err := strconv.Atoi(failureExitCodeRaw)
	if err != nil {
		log.Fatalln(err)
	}

	if err := commands.NewRootCmd().ExecuteContext(ctx); err != nil {
		log.Println(err)
		os.Exit(failureExitCode)
	}
}

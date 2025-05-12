package main

import (
	"context"
	"errors"
	"github.com/cirruslabs/gitlab-tart-executor/internal/commands"
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
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

	buildFailureExitCode := gitlabExitCode("BUILD_FAILURE_EXIT_CODE")
	systemFailureExitCode := gitlabExitCode("SYSTEM_FAILURE_EXIT_CODE")

	if err := commands.NewRootCmd().ExecuteContext(ctx); err != nil {
		log.Println(err)

		var systemFailureError *gitlab.SystemFailureError
		if errors.As(err, &systemFailureError) {
			os.Exit(systemFailureExitCode)
		}

		os.Exit(buildFailureExitCode)
	}
}

func gitlabExitCode(key string) int {
	exitCodeRaw := os.Getenv(key)
	if exitCodeRaw == "" {
		return 1
	}

	exitCode, err := strconv.Atoi(exitCodeRaw)
	if err != nil {
		log.Fatalf("failed to parse %s's value %q: %v", key, exitCodeRaw, err)
	}

	return exitCode
}

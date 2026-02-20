package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"strconv"

	"github.com/cirruslabs/gitlab-tart-executor/internal/commands"
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
)

func main() {
	// Set up signal interruptible context
	ctx, cancel := context.WithCancel(context.Background())

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)

	// Disable timestamps in logs since GitLab Runner automatically
	// adds them for us, see FF_TIMESTAMPS[1], which defaults to "true".
	//
	// [1]: https://docs.gitlab.com/runner/configuration/feature-flags/
	log.SetFlags(log.LstdFlags &^ (log.Ldate | log.Ltime))

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
		//nolint:gosec // G706 is a false-positive here: key is pre-determined and exitCodeRaw is escaped with %q
		log.Fatalf("failed to parse %s's value %q: %v", key, exitCodeRaw, err)
	}

	return exitCode
}

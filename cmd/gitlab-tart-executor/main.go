package main

import (
	"context"
	"github.com/cirruslabs/gitlab-tart-executor/internal/commands"
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"log"
	"os"
	"os/signal"
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

	env, err := gitlab.InitEnv()
	if err != nil {
		log.Fatal(err)
	}

	if err := commands.NewRootCmd().ExecuteContext(ctx); err != nil {
		log.Println(err)
		os.Exit(env.FailureExitCode)
	}
}

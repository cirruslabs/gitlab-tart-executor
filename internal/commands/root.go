package commands

import (
	"github.com/cirruslabs/gitlab-tart-executor/internal/commands/cleanup"
	"github.com/cirruslabs/gitlab-tart-executor/internal/commands/config"
	"github.com/cirruslabs/gitlab-tart-executor/internal/commands/localnetworkhelper"
	"github.com/cirruslabs/gitlab-tart-executor/internal/commands/prepare"
	"github.com/cirruslabs/gitlab-tart-executor/internal/commands/run"
	"github.com/cirruslabs/gitlab-tart-executor/internal/version"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	command := &cobra.Command{
		Use:           "executor",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.FullVersion,
	}

	command.AddCommand(
		config.NewCommand(),
		prepare.NewCommand(),
		run.NewCommand(),
		cleanup.NewCommand(),
		localnetworkhelper.NewCommand(),
	)

	return command
}

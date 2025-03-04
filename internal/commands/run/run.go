package run

import (
	"fmt"
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/spf13/cobra"
	"log"
	"os"
)

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "run <path-to-script-file>",
		Short: "Run GitLab's scripts in a Tart VM",
		RunE:  runScriptInsideVM,
		Args:  cobra.MinimumNArgs(1),
	}

	return command
}

func runScriptInsideVM(cmd *cobra.Command, args []string) error {
	scriptFile, err := os.Open(args[0])
	if err != nil {
		return err
	}

	gitLabEnv, err := gitlab.InitEnv()
	if err != nil {
		return err
	}

	vm := tart.ExistingVM(*gitLabEnv)

	// Monitor "tart run" command's output so it's not silenced
	go vm.MonitorTartRunOutput()

	config, err := tart.NewConfigFromEnvironment()
	if err != nil {
		return err
	}

	ssh, err := vm.OpenSSH(cmd.Context(), config)
	if err != nil {
		return err
	}
	defer ssh.Close()

	session, err := ssh.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	if config.KeychainUnlock {
		// This might be too spammy
		log.Println("Unlocking keychain...")

		keySession, err := ssh.NewSession()
		if err != nil {
			return err
		}
		defer keySession.Close()

		if err := keySession.Run(fmt.Sprintf("sudo security unlock-keychain -p \"%s\"", config.SSHPassword)); err != nil {
			return err
		}
	}

	// GitLab script ends with an `exit` command which will terminate the SSH session
	session.Stdin = scriptFile
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	if config.Shell != "" {
		err = session.Start(config.Shell)
	} else {
		err = session.Shell()
	}
	if err != nil {
		return err
	}

	return session.Wait()
}

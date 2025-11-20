package run

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
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

	sshClient, err := vm.OpenSSH(cmd.Context(), config)
	if err != nil {
		return err
	}
	defer sshClient.Close()

	sshSession, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer sshSession.Close()

	// GitLab script ends with an `exit` command which will terminate the SSH session
	sshSession.Stdin = scriptFile
	sshSession.Stdout = os.Stdout
	sshSession.Stderr = os.Stderr

	if config.Shell != "" {
		err = sshSession.Start(config.Shell)
	} else {
		err = sshSession.Shell()
	}
	if err != nil {
		return err
	}

	if err = sshSession.Wait(); err != nil {
		var sshExitError *ssh.ExitError
		if errors.As(err, &sshExitError) {
			propagateSSHExitError(sshExitError)
		}

		return err
	}

	return nil
}

func propagateSSHExitError(sshExitError *ssh.ExitError) {
	exitCodeFile, ok := os.LookupEnv("BUILD_EXIT_CODE_FILE")
	if !ok {
		return
	}

	//nolint:gosec // G306 shouldn't apply here as we're not writing anything sensitive
	err := os.WriteFile(exitCodeFile, fmt.Appendf(nil, "%d\n", sshExitError.ExitStatus()),
		0644)
	if err != nil {
		log.Printf("failed to propagate SSH command exit code to a file"+
			"pointed by the BUILD_EXIT_CODE_FILE environment variable (%q): %v",
			exitCodeFile, err)
	}
}

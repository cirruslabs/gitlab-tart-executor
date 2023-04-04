package run

import (
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/spf13/cobra"
	"os"
)

var config = tart.TartConfig{
	SSHUsername: "admin",
	SSHPassword: "admin",
}

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

	// GitLab script ends with an `exit` command which will terminate the SSH session
	session.Stdin = scriptFile
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	err = session.Shell()
	if err != nil {
		return err
	}

	return session.Wait()
}

package prepare

import (
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/spf13/cobra"
	"log"
)

var config = tart.TartConfig{
	SSHUsername: "admin",
	SSHPassword: "admin",
	Headless:    true,
	AlwaysPull:  true,
}

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare a Tart VM for execution",
		RunE:  runPrepareVM,
	}

	command.PersistentFlags().StringVarP(&config.SSHUsername, "username", "", config.SSHUsername, "SSH username")
	command.PersistentFlags().StringVarP(&config.SSHPassword, "password", "", config.SSHPassword, "SSH password")
	command.PersistentFlags().BoolVarP(&config.Headless, "headless", "", config.Headless, "Run VM in headless mode")
	command.PersistentFlags().BoolVarP(&config.AlwaysPull, "always-pull", "", config.AlwaysPull, "Always pull the latest version of the Tart image")
	command.PersistentFlags().BoolVarP(&config.Softnet, "softnet", "", config.Softnet, "Enable softnet")
	command.PersistentFlags().Uint64VarP(&config.CPU, "cpu", "", config.CPU, "Override default image CPU configuration")
	command.PersistentFlags().Uint64VarP(&config.Memory, "memory", "", config.Memory, "Override default image memory (in Mb) configuration")

	return command
}

func runPrepareVM(cmd *cobra.Command, args []string) error {
	gitLabEnv, err := gitlab.InitEnv()
	if err != nil {
		return err
	}
	if config.AlwaysPull {
		log.Printf("Pulling the latest version of %s...\n", gitLabEnv.JobImage)
		_, _, err := tart.TartExec(cmd.Context(), "pull", gitLabEnv.JobImage)
		if err != nil {
			return err
		}
	}

	log.Println("Cloning and configuring a new VM...")
	vm, err := tart.CreateNewVM(cmd.Context(), *gitLabEnv, config)
	if err != nil {
		return err
	}
	err = vm.Start(config)
	if err != nil {
		return err
	}
	log.Println("Waiting for the VM to boot and be SSH-able...")
	ssh, err := vm.OpenSSH(cmd.Context(), config)
	if err != nil {
		return err
	}
	log.Println("Was able to SSH! VM is ready.")

	return ssh.Close()
}

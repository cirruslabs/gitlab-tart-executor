package prepare

import (
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var cpuOverride uint64
var memoryOverride uint64

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare a Tart VM for execution",
		RunE:  runPrepareVM,
	}

	command.PersistentFlags().Uint64Var(&cpuOverride, "cpu", 0,
		"Override default image CPU configuration (number of CPUs)")
	command.PersistentFlags().Uint64Var(&memoryOverride, "memory", 0,
		"Override default image memory configuration (size in megabytes)")

	return command
}

func runPrepareVM(cmd *cobra.Command, args []string) error {
	gitLabEnv, err := gitlab.InitEnv()
	if err != nil {
		return err
	}

	config, err := tart.NewConfigFromEnvironment()
	if err != nil {
		return err
	}

	if config.AlwaysPull {
		log.Printf("Pulling the latest version of %s...\n", gitLabEnv.JobImage)
		_, _, err := tart.TartExecWithEnv(cmd.Context(), additionalPullEnv(gitLabEnv.Registry),
			"pull", gitLabEnv.JobImage)
		if err != nil {
			return err
		}
	}

	log.Println("Cloning and configuring a new VM...")
	vm, err := tart.CreateNewVM(cmd.Context(), *gitLabEnv, cpuOverride, memoryOverride)
	if err != nil {
		return err
	}
	err = vm.Start(config, gitLabEnv)
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

func additionalPullEnv(registry *gitlab.Registry) map[string]string {
	// Prefer manual registry credentials override from the user
	tartRegistryUsername, tartRegistryUsernameOK := os.LookupEnv("CUSTOM_ENV_TART_REGISTRY_USERNAME")
	tartRegistryPassword, tartRegistryPasswordOK := os.LookupEnv("CUSTOM_ENV_TART_REGISTRY_PASSWORD")
	if tartRegistryUsernameOK && tartRegistryPasswordOK {
		return map[string]string{
			"TART_REGISTRY_USERNAME": tartRegistryUsername,
			"TART_REGISTRY_PASSWORD": tartRegistryPassword,
		}
	}

	// Otherwise fallback to GitLab's provided registry credentials, if any
	if registry != nil {
		return map[string]string{
			"TART_REGISTRY_USERNAME": registry.User,
			"TART_REGISTRY_PASSWORD": registry.Password,
		}
	}

	return nil
}

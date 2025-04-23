package cleanup

import (
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/spf13/cobra"
	"log"
	"os"
)

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup Tart VM after job finishes",
		RunE:  cleanupVM,
	}

	return command
}

func cleanupVM(cmd *cobra.Command, args []string) error {
	gitLabEnv, err := gitlab.InitEnv()
	if err != nil {
		return err
	}

	vm := tart.ExistingVM(*gitLabEnv)

	if err = vm.Stop(); err != nil {
		log.Printf("Failed to stop VM: %v", err)
	}

	if err := vm.Delete(); err != nil {
		log.Printf("Failed to delete VM: %v", err)

		return err
	}

	tartConfig, err := tart.NewConfigFromEnvironment(gitlab.EnvPrefixGitLabRunner)
	if err != nil {
		return err
	}

	if tartConfig.HostDir {
		if err := os.RemoveAll(gitLabEnv.HostDirPath()); err != nil {
			log.Printf("Failed to clean up %q (temporary directory from the host): %v",
				gitLabEnv.HostDirPath(), err)

			return err
		}
	}

	return nil
}

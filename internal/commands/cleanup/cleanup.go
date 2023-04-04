package cleanup

import (
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/spf13/cobra"
	"log"
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

	err = vm.Stop()
	if err != nil {
		log.Printf("Failed to stop VM: %v", err)
	}
	return vm.Delete()
}

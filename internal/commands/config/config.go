package config

import (
	"encoding/json"
	"fmt"
	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/spf13/cobra"
	"os"
)

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "config",
		Short: "Configure GitLab Runner",
		RunE:  runConfig,
	}

	return command
}

func runConfig(cmd *cobra.Command, args []string) error {
	gitlabRunnerConfig := struct {
		BuildsDir string            `json:"builds_dir"`
		CacheDir  string            `json:"cache_dir"`
		JobEnv    map[string]string `json:"job_env,omitempty"`
	}{
		BuildsDir: "builds",
		CacheDir:  "cache",
		JobEnv:    map[string]string{},
	}

	tartConfig, err := tart.NewConfigFromEnvironment()
	if err != nil {
		return err
	}

	if tartConfig.HostDir {
		gitlabRunnerConfig.BuildsDir = "/Volumes/My Shared Files/hostdir"

		tmpDir, err := os.MkdirTemp("", "")
		if err != nil {
			return err
		}

		gitlabRunnerConfig.JobEnv["TART_EXECUTOR_INTERNAL_HOST_DIR_PATH"] = tmpDir
	}

	jsonBytes, err := json.MarshalIndent(&gitlabRunnerConfig, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonBytes))

	return nil
}

package config

import (
	"encoding/json"
	"fmt"
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/spf13/cobra"
	"os"
)

var cacheDir string

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "config",
		Short: "Configure GitLab Runner",
		RunE:  runConfig,
	}

	command.PersistentFlags().StringVar(&cacheDir, "cache-dir", "",
		"path to a directory on host to use for caching purposes")

	return command
}

func runConfig(cmd *cobra.Command, args []string) error {
	gitlabRunnerConfig := struct {
		BuildsDir string            `json:"builds_dir"`
		CacheDir  string            `json:"cache_dir"`
		JobEnv    map[string]string `json:"job_env,omitempty"`
	}{
		// 1. GitLab Runner's documentation requires the builds and cache directory paths
		// to be absolute[1].
		//
		// 2. GitLab Runner uses relative paths internally which results in improper directory traversal[2],
		// this is why we use "/private/tmp" instead of just "/tmp" here as a workaround.
		//
		// [1]: https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runners-section
		// [2]: https://gitlab.com/gitlab-org/gitlab-runner/-/issues/31003
		BuildsDir: "/private/tmp/builds",
		CacheDir:  "/private/tmp/cache",
		JobEnv:    map[string]string{},
	}

	tartConfig, err := tart.NewConfigFromEnvironment()
	if err != nil {
		return err
	}

	gitLabEnv, err := gitlab.InitEnv()
	if err != nil {
		return err
	}

	if tartConfig.HostDir {
		gitlabRunnerConfig.BuildsDir = "/Volumes/My Shared Files/hostdir"

		if err := os.MkdirAll(gitLabEnv.HostDirPath(), 0700); err != nil {
			return err
		}
	}

	if cacheDir != "" {
		gitlabRunnerConfig.CacheDir = "/Volumes/My Shared Files/cachedir"
		gitlabRunnerConfig.JobEnv[tart.EnvTartExecutorInternalCacheDir] = cacheDir
	}

	jsonBytes, err := json.MarshalIndent(&gitlabRunnerConfig, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonBytes))

	return nil
}

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/spf13/cobra"
	"os"
)

var ErrConfigFailed = errors.New("configuration stage failed")

var (
	buildsDir string
	cacheDir  string

	guestBuildsDir string
	guestCacheDir  string
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure GitLab Runner",
		RunE:  runConfig,
	}

	cmd.PersistentFlags().StringVar(&buildsDir, "builds-dir", "",
		"path to a directory on host to use for storing builds, automatically mounts that directory "+
			"to the guest VM, mutually exclusive with \"--guest-builds-dir\"")
	cmd.PersistentFlags().StringVar(&cacheDir, "cache-dir", "",
		"path to a directory on host to use for caching purposes, automatically mounts that directory "+
			"to the guest VM, mutually exclusive with \"--guest-cache-dir\"")
	cmd.PersistentFlags().StringVar(&guestBuildsDir, "guest-builds-dir", "",
		"path to a directory in guest to use for storing builds, useful when mounting a block device "+
			"via \"--disk\" command-line argument (mutually exclusive with \"--builds-dir\")")
	cmd.PersistentFlags().StringVar(&guestCacheDir, "guest-cache-dir", "",
		"path to a directory in guest to use for caching purposes, useful when mounting a block device "+
			"via \"--disk\" command-line argument (mutually exclusive with \"--cache-dir\")")

	return cmd
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

	// Validate environment variables and command-line arguments combinations
	if tartConfig.HostDir && buildsDir != "" {
		return fmt.Errorf("%w: --builds-dir and TART_EXECUTOR_HOST_DIR are mutually exclusive",
			ErrConfigFailed)
	}
	if tartConfig.HostDir && guestBuildsDir != "" {
		return fmt.Errorf("%w: --guest-builds-dir and TART_EXECUTOR_HOST_DIR are mutually exclusive",
			ErrConfigFailed)
	}

	if buildsDir != "" && guestBuildsDir != "" {
		return fmt.Errorf("%w: --builds-dir and --guest-builds-dir are mutually exclusive",
			ErrConfigFailed)
	}

	if cacheDir != "" && guestCacheDir != "" {
		return fmt.Errorf("%w: --cache-dir and --guest-cache-dir are mutually exclusive",
			ErrConfigFailed)
	}

	// Figure out the builds directory override to use
	switch {
	case tartConfig.HostDir:
		gitlabRunnerConfig.BuildsDir = "/Volumes/My Shared Files/hostdir"

		if err := os.MkdirAll(gitLabEnv.HostDirPath(), 0700); err != nil {
			return err
		}
	case buildsDir != "":
		gitlabRunnerConfig.BuildsDir = "/Volumes/My Shared Files/buildsdir"
		buildsDir = os.ExpandEnv(buildsDir)
		gitlabRunnerConfig.JobEnv[tart.EnvTartExecutorInternalBuildsDir] = buildsDir

		if err := os.MkdirAll(buildsDir, 0700); err != nil {
			return err
		}
	case guestBuildsDir != "":
		gitlabRunnerConfig.BuildsDir = guestBuildsDir
	}

	// Figure out the cache directory override to use
	switch {
	case cacheDir != "":
		gitlabRunnerConfig.CacheDir = "/Volumes/My Shared Files/cachedir"
		cacheDir = os.ExpandEnv(cacheDir)
		gitlabRunnerConfig.JobEnv[tart.EnvTartExecutorInternalCacheDir] = cacheDir

		if err := os.MkdirAll(cacheDir, 0700); err != nil {
			return err
		}
	case guestCacheDir != "":
		gitlabRunnerConfig.CacheDir = guestCacheDir
	}

	jsonBytes, err := json.MarshalIndent(&gitlabRunnerConfig, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonBytes))

	return nil
}

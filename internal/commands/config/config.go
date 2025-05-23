package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/cirruslabs/gitlab-tart-executor/internal/version"
	"github.com/spf13/cobra"
	"os"
)

const (
	driverName    = "name"
	driverVersion = "version"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runConfig(cmd, args); err != nil {
				return gitlab.NewSystemFailureError(err)
			}

			return nil
		},
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

func runConfig(_ *cobra.Command, _ []string) error {
	gitlabRunnerDriver := map[string]string{
		driverName:    tart.TartCommandName,
		driverVersion: version.FullVersion,
	}
	gitlabRunnerConfig := struct {
		BuildsDir string            `json:"builds_dir"`
		CacheDir  string            `json:"cache_dir"`
		JobEnv    map[string]string `json:"job_env,omitempty"`
		Driver    map[string]string `json:"driver,omitempty"`
	}{
		// 1. GitLab Runner's documentation requires the builds and cache directory paths
		// to be absolute[1].
		//
		// 2. GitLab Runner uses relative paths internally which results in improper directory traversal[2],
		// so instead of "/tmp" we need to use "/private/tmp" here as a workaround.
		//
		// 3. However, there's no "/private/tmp" on Linux. So we use the lowest common denominator
		// in the form of "/var/tmp". It's both (1) not a symbolic link and (2) is present on both platforms.
		//
		// [1]: https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runners-section
		// [2]: https://gitlab.com/gitlab-org/gitlab-runner/-/issues/31003
		BuildsDir: "/var/tmp/builds",
		CacheDir:  "/var/tmp/cache",
		Driver:    gitlabRunnerDriver,
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
		gitlabRunnerConfig.JobEnv[tart.EnvTartExecutorInternalBuildsDirOnHost] = gitLabEnv.HostDirPath()

		if err := os.MkdirAll(gitLabEnv.HostDirPath(), 0700); err != nil {
			return err
		}
	case buildsDir != "":
		buildsDir = os.ExpandEnv(buildsDir)
		gitlabRunnerConfig.JobEnv[tart.EnvTartExecutorInternalBuildsDirOnHost] = buildsDir

		if err := os.MkdirAll(buildsDir, 0700); err != nil {
			return err
		}
	case guestBuildsDir != "":
		gitlabRunnerConfig.BuildsDir = guestBuildsDir
	}

	// Figure out the cache directory override to use
	switch {
	case cacheDir != "":
		cacheDir = os.ExpandEnv(cacheDir)
		gitlabRunnerConfig.JobEnv[tart.EnvTartExecutorInternalCacheDirOnHost] = cacheDir

		if err := os.MkdirAll(cacheDir, 0700); err != nil {
			return err
		}
	case guestCacheDir != "":
		gitlabRunnerConfig.CacheDir = guestCacheDir
	}

	// Propagate builds and cache directory locations in the guest
	// because GitLab Runner won't do this for us
	gitlabRunnerConfig.JobEnv[tart.EnvTartExecutorInternalBuildsDir] = gitlabRunnerConfig.BuildsDir
	gitlabRunnerConfig.JobEnv[tart.EnvTartExecutorInternalCacheDir] = gitlabRunnerConfig.CacheDir

	jsonBytes, err := json.MarshalIndent(&gitlabRunnerConfig, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonBytes))

	return nil
}

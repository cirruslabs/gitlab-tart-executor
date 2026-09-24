package tart

import (
	"errors"
	"fmt"
	"os"

	"github.com/caarlos0/env/v8"
)

var ErrConfigFromEnvironmentFailed = errors.New("failed to load config from environment")

const (
	// GitLab CI/CD environment adds "CUSTOM_ENV_" prefix[1] to prevent
	// conflicts with system environment variables.
	//
	// [1]: https://docs.gitlab.com/runner/executors/custom.html#stages
	envPrefixGitLabRunner = "CUSTOM_ENV_"

	// The prefix that we use to avoid confusion with Cirrus CI Cloud variables
	// and remove repetition from the Config's struct declaration.
	envPrefixTartExecutor = "TART_EXECUTOR_"

	// EnvTartExecutorInternalBuildsDir is an internal environment variable
	// that does not use the "CUSTOM_ENV_" prefix, thus preventing the override
	// by the user.
	EnvTartExecutorInternalBuildsDir = "TART_EXECUTOR_INTERNAL_BUILDS_DIR"

	// EnvTartExecutorInternalBuildsDirOnHost is an internal environment variable
	// that does not use the "CUSTOM_ENV_" prefix, thus preventing the override
	// by the user.
	EnvTartExecutorInternalBuildsDirOnHost = "TART_EXECUTOR_INTERNAL_BUILDS_DIR_ON_HOST"

	// EnvTartExecutorInternalCacheDir is an internal environment variable
	// that does not use the "CUSTOM_ENV_" prefix, thus preventing the override
	// by the user.
	EnvTartExecutorInternalCacheDir = "TART_EXECUTOR_INTERNAL_CACHE_DIR"

	// EnvTartExecutorInternalCacheDirOnHost is an internal environment variable
	// that does not use the "CUSTOM_ENV_" prefix, thus preventing the override
	// by the user.
	EnvTartExecutorInternalCacheDirOnHost = "TART_EXECUTOR_INTERNAL_CACHE_DIR_ON_HOST"

	// EnvTartExecutorInternalSSHPassword is an internal environment variable
	// that does not use the "CUSTOM_ENV_" prefix, thus preventing the override
	// by the user.
	EnvTartExecutorInternalSSHPassword = "TART_EXECUTOR_INTERNAL_SSH_PASSWORD"
)

type Config struct {
	SSHUsername         string `env:"SSH_USERNAME" envDefault:"admin"`
	SSHPassword         string `env:"SSH_PASSWORD" envDefault:"admin"`
	SSHPort             uint16 `env:"SSH_PORT" envDefault:"22"`
	Bridged             string `env:"BRIDGED"`
	Softnet             bool   `env:"SOFTNET"`
	SoftnetAllow        string `env:"SOFTNET_ALLOW"`
	Headless            bool   `env:"HEADLESS"  envDefault:"true"`
	RandomMAC           bool   `env:"RANDOM_MAC"  envDefault:"true"`
	RootDiskOpts        string `env:"ROOT_DISK_OPTS"`
	AlwaysPull          bool   `env:"ALWAYS_PULL"  envDefault:"true"`
	InsecurePull        bool   `env:"INSECURE_PULL"  envDefault:"false"`
	PullConcurrency     uint8  `env:"PULL_CONCURRENCY"`
	HostDir             bool   `env:"HOST_DIR"`
	Shell               string `env:"SHELL"`
	InstallGitlabRunner string `env:"INSTALL_GITLAB_RUNNER"`
	Timezone            string `env:"TIMEZONE"`
	Display             string `env:"DISPLAY"`
}

func NewConfigFromEnvironment() (Config, error) {
	var config Config

	if err := env.ParseWithOptions(&config, env.Options{
		Prefix: envPrefixGitLabRunner + envPrefixTartExecutor,
	}); err != nil {
		return config, fmt.Errorf("%w: %v", ErrConfigFromEnvironmentFailed, err)
	}

	// Three-tier precedence for the SSH password:
	//
	//   1. CI/CD variable (CUSTOM_ENV_TART_EXECUTOR_SSH_PASSWORD) — backward
	//      compatible, but leaks into the job's environment.
	//
	//   2. Runner-side internal variable (TART_EXECUTOR_INTERNAL_SSH_PASSWORD)
	//      set by the config stage's --default-ssh-password flag — doesn't leak.
	//
	//   3. Built-in default ("admin").
	if _, ciCdSet := os.LookupEnv(envPrefixGitLabRunner + envPrefixTartExecutor + "SSH_PASSWORD"); ciCdSet {
		// The env.ParseWithOptions call above already parsed the CI/CD variable,
		// so config.SSHPassword is correct — nothing to do.
	} else if internalPassword, ok := os.LookupEnv(EnvTartExecutorInternalSSHPassword); ok {
		config.SSHPassword = internalPassword
	}

	return config, nil
}

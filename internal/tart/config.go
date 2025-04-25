package tart

import (
	"errors"
	"fmt"
	"github.com/caarlos0/env/v8"
)

var ErrConfigFromEnvironmentFailed = errors.New("failed to load config from environment")

const (
	// The prefix that we use to avoid confusion with Cirrus CI Cloud variables
	// and remove repetition from the Config's struct declaration.
	EnvPrefixTartExecutor = "TART_EXECUTOR_"

	// EnvTartExecutorInternalBuildsDir is an internal environment variable
	// that does not use the "CUSTOM_ENV_" prefix, thus preventing the override
	// by the user.
	EnvTartExecutorInternalBuildsDir = EnvPrefixTartExecutor + "INTERNAL_BUILDS_DIR"

	// EnvTartExecutorInternalBuildsDirOnHost is an internal environment variable
	// that does not use the "CUSTOM_ENV_" prefix, thus preventing the override
	// by the user.
	EnvTartExecutorInternalBuildsDirOnHost = EnvPrefixTartExecutor + "INTERNAL_BUILDS_DIR_ON_HOST"

	// EnvTartExecutorInternalCacheDir is an internal environment variable
	// that does not use the "CUSTOM_ENV_" prefix, thus preventing the override
	// by the user.
	EnvTartExecutorInternalCacheDir = EnvPrefixTartExecutor + "INTERNAL_CACHE_DIR"

	// EnvTartExecutorInternalCacheDirOnHost is an internal environment variable
	// that does not use the "CUSTOM_ENV_" prefix, thus preventing the override
	// by the user.
	EnvTartExecutorInternalCacheDirOnHost = EnvPrefixTartExecutor + "INTERNAL_CACHE_DIR_ON_HOST"

	// EnvTartRegistry is the environment variable containing the authentication
	// principal for the image registry.
	EnvTartRegistryUsername = "TART_REGISTRY_USERNAME"

	// EnvTartRegistry is the environment variable containing the authentication
	// secret for the image registry.
	EnvTartRegistryPassword = "TART_REGISTRY_PASSWORD" //nolint: gosec

	// EnvTartRegistry is the environment variable containing the address
	// of the image registry.
	EnvTartRegistryHostname = "TART_REGISTRY_HOSTNAME"
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
}

func NewConfigFromEnvironment(prefix string) (Config, error) {
	var config Config

	if err := env.ParseWithOptions(&config, env.Options{
		Prefix: prefix + EnvPrefixTartExecutor,
	}); err != nil {
		return config, fmt.Errorf("%w: %v", ErrConfigFromEnvironmentFailed, err)
	}

	return config, nil
}

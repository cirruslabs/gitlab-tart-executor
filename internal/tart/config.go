package tart

import (
	"errors"
	"fmt"
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
	envPrefixGitlabTartExecutor = "TART_EXECUTOR_"
)

type Config struct {
	SSHUsername         string `env:"SSH_USERNAME" envDefault:"admin"`
	SSHPassword         string `env:"SSH_PASSWORD" envDefault:"admin"`
	Softnet             bool   `env:"SOFTNET"`
	Headless            bool   `env:"HEADLESS"  envDefault:"true"`
	AlwaysPull          bool   `env:"ALWAYS_PULL"  envDefault:"true"`
	HostDir             bool   `env:"HOST_DIR"`
	Shell               string `env:"SHELL"`
	InstallGitlabRunner bool   `env:"INSTALL_GITLAB_RUNNER"`
}

func NewConfigFromEnvironment() (Config, error) {
	var config Config

	if err := env.ParseWithOptions(&config, env.Options{
		Prefix: envPrefixGitLabRunner + envPrefixGitlabTartExecutor,
	}); err != nil {
		return config, fmt.Errorf("%w: %v", ErrConfigFromEnvironmentFailed, err)
	}

	return config, nil
}

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
	gitlabRunnerPrefix = "CUSTOM_ENV_"

	// The prefix that we use to avoid confusion with Cirrus CI Cloud variables
	// and remove repetition from the Config's struct declaration.
	cirrusGitlabTartExecutorPrefix = "CIRRUS_GTE_"
)

type Config struct {
	SSHUsername string `env:"SSH_USERNAME" envDefault:"admin"`
	SSHPassword string `env:"SSH_PASSWORD" envDefault:"admin"`
	CPU         uint64 `env:"CPU"`
	Memory      uint64 `env:"MEMORY"`
	Softnet     bool   `env:"SOFTNET"`
	Headless    bool   `env:"HEADLESS"  envDefault:"true"`
	AlwaysPull  bool   `env:"ALWAYS_PULL"  envDefault:"true"`
}

func NewConfigFromEnvironment() (Config, error) {
	var config Config

	if err := env.ParseWithOptions(&config, env.Options{
		Prefix: gitlabRunnerPrefix + cirrusGitlabTartExecutorPrefix,
	}); err != nil {
		return config, fmt.Errorf("%w: %v", ErrConfigFromEnvironmentFailed, err)
	}

	return config, nil
}

package tart

import (
	"errors"
	"fmt"
	"github.com/caarlos0/env/v8"
)

var ErrInternalConfigFromEnvironmentFailed = errors.New("failed to load internal config from environment")

const (
	// The prefix that we use to avoid confusion with user-facing
	// GitLab Tart Executor environment variables.
	envPrefixGitlabTartExecutorInternal = "INTERNAL_"
)

type InternalConfig struct {
	HostDirPath string `env:"HOST_DIR_PATH"`
}

func NewInternalConfigFromEnvironment() (InternalConfig, error) {
	var config InternalConfig

	if err := env.ParseWithOptions(&config, env.Options{
		Prefix: envPrefixGitlabTartExecutor + envPrefixGitlabTartExecutorInternal,
	}); err != nil {
		return config, fmt.Errorf("%w: %v", ErrInternalConfigFromEnvironmentFailed, err)
	}

	return config, nil
}

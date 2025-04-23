package gitlab

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// EnvPrefixGitLabRunner is the environment variable prefix
	// GitLab CI/CD adds [1] to prevent conflicts with system
	// environment variables.
	//
	// [1]: https://docs.gitlab.com/runner/executors/custom.html#stages
	EnvPrefixGitLabRunner = "CUSTOM_ENV_"
)

var ErrGitLabEnv = errors.New("GitLab environment error")

type Env struct {
	JobID           string
	JobImage        string
	FailureExitCode int
	Registry        *Registry
}

type Registry struct {
	Address  string
	User     string
	Password string
}

func (e Env) VirtualMachineID() string {
	return fmt.Sprintf("gitlab-%s", e.JobID)
}

func (e Env) HostDirPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("tart-executor-host-dir-%s", e.JobID))
}

func InitEnv() (*Env, error) {
	result := &Env{}
	jobID, ok := os.LookupEnv("CUSTOM_ENV_CI_JOB_ID")
	if !ok {
		return nil, fmt.Errorf("%w: CUSTOM_ENV_CI_JOB_ID is missing", ErrGitLabEnv)
	}

	result.JobID = jobID
	result.JobImage = os.Getenv("CUSTOM_ENV_CI_JOB_IMAGE")

	failureExitCodeRaw := os.Getenv("BUILD_FAILURE_EXIT_CODE")
	if failureExitCodeRaw == "" {
		failureExitCodeRaw = "1" // default value
	}

	failureExitCode, err := strconv.Atoi(failureExitCodeRaw)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse BUILD_FAILURE_EXIT_CODE", ErrGitLabEnv)
	}
	result.FailureExitCode = failureExitCode

	ciRegistry, ciRegistryOK := os.LookupEnv("CUSTOM_ENV_CI_REGISTRY")
	ciRegistryUser, ciRegistryUserOK := os.LookupEnv("CUSTOM_ENV_CI_REGISTRY_USER")
	ciRegistryPassword, ciRegistryPasswordOK := os.LookupEnv("CUSTOM_ENV_CI_REGISTRY_PASSWORD")
	if ciRegistryOK && ciRegistryUserOK && ciRegistryPasswordOK {
		result.Registry = &Registry{
			Address:  ciRegistry,
			User:     ciRegistryUser,
			Password: ciRegistryPassword,
		}
	}

	return result, nil
}

// CustomExecutorEnvironment parses the provided
// environment variables for values intended for the custom
// executor based on their prefix. The resulting list items
// have their prefix stripped.
func CustomExecutorEnvironment(env []string) []string {
	vars := make([]string, 0, len(env))

	for _, e := range env {
		pair := strings.TrimPrefix(e, EnvPrefixGitLabRunner)
		if pair != "" && pair[0] != '=' && pair != e {
			vars = append(vars, pair)
		}
	}

	return vars
}

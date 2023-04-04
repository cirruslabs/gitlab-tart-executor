package gitlab

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

var ErrGitLabEnv = errors.New("GitLab environment error")

type Env struct {
	JobID           string
	JobImage        string
	FailureExitCode int
}

func (e Env) VirtualMachineID() string {
	return fmt.Sprintf("gitlab-%s", e.JobID)
}

func InitEnv() (*Env, error) {
	result := &Env{}
	jobID, ok := os.LookupEnv("CUSTOM_ENV_CI_JOB_ID")
	if !ok {
		return nil, fmt.Errorf("%w: CUSTOM_ENV_CI_JOB_ID is missing", ErrGitLabEnv)
	}
	result.JobID = jobID
	jobImage, ok := os.LookupEnv("CUSTOM_ENV_CI_JOB_IMAGE")
	if !ok {
		return nil, fmt.Errorf("%w: CUSTOM_ENV_CI_JOB_IMAGE is missing", ErrGitLabEnv)
	}
	result.JobImage = jobImage
	failureExitCodeRaw := os.Getenv("BUILD_FAILURE_EXIT_CODE")
	if failureExitCodeRaw == "" {
		failureExitCodeRaw = "1" // default value
	}
	failureExitCode, err := strconv.Atoi(failureExitCodeRaw)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse BUILD_FAILURE_EXIT_CODE", ErrGitLabEnv)
	}
	result.FailureExitCode = failureExitCode
	return result, nil
}

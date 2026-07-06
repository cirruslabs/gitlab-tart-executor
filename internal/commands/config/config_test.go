package config

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/stretchr/testify/require"
)

func TestConfigHostDirWithGuestBuildsDir(t *testing.T) {
	envHostDir := "CUSTOM_ENV_TART_EXECUTOR_HOST_DIR"
	envJobID := "CUSTOM_ENV_CI_JOB_ID"
	guestBuildsDirValue := "/custom/guest/builds"

	savedHostDir, hadHostDir := os.LookupEnv(envHostDir)
	savedJobID, hadJobID := os.LookupEnv(envJobID)
	savedGuestBuildsDir := guestBuildsDir

	os.Setenv(envHostDir, "true")
	os.Setenv(envJobID, "test-123")
	guestBuildsDir = guestBuildsDirValue

	defer func() {
		os.Unsetenv(envHostDir)
		os.Unsetenv(envJobID)
		if hadHostDir {
			os.Setenv(envHostDir, savedHostDir)
		}
		if hadJobID {
			os.Setenv(envJobID, savedJobID)
		}
		guestBuildsDir = savedGuestBuildsDir
	}()

	r, w, err := os.Pipe()
	require.NoError(t, err)
	tmp := os.Stdout
	os.Stdout = w
	defer func() {
		os.Stdout = tmp
	}()

	err = runConfig(nil, nil)
	w.Close()
	require.NoError(t, err)

	out, err := io.ReadAll(r)
	require.NoError(t, err)

	var result struct {
		BuildsDir string            `json:"builds_dir"`
		JobEnv    map[string]string `json:"job_env"`
	}
	require.NoError(t, json.Unmarshal(out, &result))
	require.Equal(t, guestBuildsDirValue, result.BuildsDir)

	require.Contains(t, result.JobEnv, tart.EnvTartExecutorInternalBuildsDirOnHost)
	require.Contains(t, result.JobEnv, tart.EnvTartExecutorInternalBuildsDir)
	require.Equal(t, guestBuildsDirValue, result.JobEnv[tart.EnvTartExecutorInternalBuildsDir])
}

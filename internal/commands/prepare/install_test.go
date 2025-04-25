package prepare //nolint:testpackage

import (
	"io"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestInstallGitlabRunnerScript(t *testing.T) {
	tests := map[string]struct {
		WantError   bool
		WantFixture string
		Options     []templateOption
		Method      string
	}{
		"auto detect, latest release": {
			WantFixture: "install-gitlab-runner-auto.sh",
			Options: []templateOption{
				withGitlabRunnerInstaller("true"),
			},
		},
		"homebrew, latest release": {
			WantFixture: "install-gitlab-runner-brew.sh",
			Options: []templateOption{
				withGitlabRunnerInstaller("brew"),
			},
		},
		"curl, latest release": {
			WantFixture: "install-gitlab-runner-curl.sh",
			Options: []templateOption{
				withGitlabRunnerInstaller("curl"),
			},
		},
		"curl, specific version": {
			WantFixture: "install-gitlab-runner-version.sh",
			Options: []templateOption{
				withGitlabRunnerInstaller("17.10.0"),
			},
		},
		"curl, invalid version": {
			WantError: true,
			Options: []templateOption{
				withGitlabRunnerInstaller("invalid-semver"),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := installGitlabRunnerScript(test.Options...)

			if test.WantError {
				assert.Assert(t, err != nil)

				return
			} else {
				assert.NilError(t, err)
			}

			actual, err := io.ReadAll(got)
			assert.NilError(t, err)
			golden.Assert(t, string(actual), test.WantFixture)
		})
	}
}

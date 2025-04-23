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
		Method      string
		Vars        []string
	}{
		"omit installation": {
			Method: "",
			Vars:   []string{},
		},
		"auto detect, latest release": {
			WantFixture: "install-gitlab-runner-auto.sh",
			Method:      "true",
			Vars:        []string{"CUSTOM_ENV_NO_PROXY=.example.com", "CUSTOM_ENV_HTTP_PROXY=http://proxy.example.com:3128"},
		},
		"homebrew, latest release": {
			WantFixture: "install-gitlab-runner-brew.sh",
			Method:      "brew",
			Vars:        []string{"CUSTOM_ENV_HOMEBREW_ARTIFACT_DOMAIN=https://mirror.example.com"},
		},
		"curl, latest release": {
			WantFixture: "install-gitlab-runner-curl.sh",
			Method:      "curl",
			Vars:        []string{"CUSTOM_ENV_NO_PROXY=.example.com", "CUSTOM_ENV_HTTP_PROXY=http://proxy.example.com:3128"},
		},
		"curl, specific version": {
			WantFixture: "install-gitlab-runner-version.sh",
			Method:      "17.10.0",
			Vars:        []string{"CUSTOM_ENV_NO_PROXY=.example.com", "CUSTOM_ENV_HTTP_PROXY=http://proxy.example.com:3128"},
		},
		"curl, invalid version": {
			WantError: true,
			Method:    "invalid-semver",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := installGitlabRunnerScript(test.Method, test.Vars)

			if test.WantError {
				assert.Assert(t, err != nil)

				return
			} else {
				assert.NilError(t, err)
			}

			if test.WantFixture == "" {
				assert.Assert(t, got == nil, "No GitLab Runner installation selected")

				return
			} else {
				assert.Assert(t, got != nil, "No GitLab Runner installation available")
			}

			actual, err := io.ReadAll(got)
			assert.NilError(t, err)
			golden.Assert(t, string(actual), test.WantFixture)
		})
	}
}

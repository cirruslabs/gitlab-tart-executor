package gitlab_test

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
)

func TestCustomExecutorEnvironment(t *testing.T) {
	have := []string{
		"",
		"CUSTOM_ENV_",
		"CUSTOM_ENV_=",
		"CUSTOM_ENV_EMPTY",
		"HOSTNAME_CUSTOM_ENV_FOO=hostname",
		"CUSTOM_ENV_FOO=bar",
		"USER=root",
	}
	got := gitlab.CustomExecutorEnvironment(have)

	assert.Assert(t, cmp.Len(got, 2))
	assert.Assert(t, cmp.Contains(got, "EMPTY"))
	assert.Assert(t, cmp.Contains(got, "FOO=bar"))
}

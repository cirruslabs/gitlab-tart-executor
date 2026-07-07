package tart_test

import (
	"os"
	"testing"

	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/stretchr/testify/require"
)

func TestNewConfigFromEnvironment_SSHPasswordPrecedence(t *testing.T) {
	envPrefix := "CUSTOM_ENV_TART_EXECUTOR_"
	envCI := envPrefix + "SSH_PASSWORD"
	envInternal := "TART_EXECUTOR_INTERNAL_SSH_PASSWORD"

	tests := []struct {
		name     string
		ciValue  string
		intValue string
		want     string
	}{
		{
			name: "ci_variable_takes_precedence_over_internal",
			ciValue: "from-ci",
			intValue: "from-internal",
			want: "from-ci",
		},
		{
			name: "internal_used_when_ci_not_set",
			ciValue: "",
			intValue: "from-internal",
			want: "from-internal",
		},
		{
			name: "default_used_when_nothing_set",
			ciValue: "",
			intValue: "",
			want: "admin",
		},
		{
			name: "ci_variable_without_internal",
			ciValue: "from-ci-only",
			intValue: "",
			want: "from-ci-only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			savedCI, hadCI := os.LookupEnv(envCI)
			savedInternal, hadInternal := os.LookupEnv(envInternal)

			os.Unsetenv(envCI)
			os.Unsetenv(envInternal)
			if tt.ciValue != "" {
				os.Setenv(envCI, tt.ciValue)
			}
			if tt.intValue != "" {
				os.Setenv(envInternal, tt.intValue)
			}

			defer func() {
				os.Unsetenv(envCI)
				os.Unsetenv(envInternal)
				if hadCI {
					os.Setenv(envCI, savedCI)
				}
				if hadInternal {
					os.Setenv(envInternal, savedInternal)
				}
			}()

			config, err := tart.NewConfigFromEnvironment()
			require.NoError(t, err)
			require.Equal(t, tt.want, config.SSHPassword)
		})
	}
}

package prepare

import (
	"fmt"
	"log"

	"github.com/Masterminds/semver/v3"
)

type templateConfig struct {
	PackageManager       string
	GitlabRunnerVersion  string
	GitlabRunnerProvider string
}

type templateOption func(*templateConfig) error

func withGitlabRunnerInstaller(method string) templateOption {
	return func(c *templateConfig) error {
		switch method {
		case "brew", "curl":
			c.PackageManager = method
		case "true", "yes", "on":
			log.Printf("%q value for TART_EXECUTOR_INSTALL_GITLAB_RUNNER will deprecated "+
				"in next version, please use either \"brew\", \"curl\" or \"major.minor.patch\"",
				method)

			c.PackageManager = ""
		default:
			version, err := semver.NewVersion(method)
			if err != nil {
				return fmt.Errorf("%w: TART_EXECUTOR_INSTALL_GITLAB_RUNNER only accepts "+
					"\"brew\", \"curl\" or \"major.minor.patch\", got %q", ErrFailed, method)
			}

			c.PackageManager = "curl"
			c.GitlabRunnerVersion = "v" + version.String()
		}

		return nil
	}
}

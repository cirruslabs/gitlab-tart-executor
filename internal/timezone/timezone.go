package timezone

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var tzRegexp = regexp.MustCompile("^[a-zA-Z-_]+(/[a-zA-Z-_]+)*$")

var ErrParseFailed = errors.New("failed to parse timezone")

func Parse(timezone string) (string, error) {
	if timezone == "auto" {
		localtimePointsTo, err := os.Readlink("/etc/localtime")
		if err != nil {
			return "", fmt.Errorf("%w: while reading /etc/localtime: %v", ErrParseFailed, err)
		}

		timezone = strings.TrimPrefix(localtimePointsTo, "/var/db/timezone/zoneinfo/")

		// UTC is not supported by systemsetup(8),
		// but the /etc/localtime can link to it
		// (see /var/db/timezone/zoneinfo directory
		// contents), so work around this.
		if timezone == "UTC" {
			timezone = "GMT"
		}
	}

	if !tzRegexp.MatchString(timezone) {
		return "", fmt.Errorf("%w: doesn't match the regular expression %s",
			ErrParseFailed, tzRegexp.String())
	}

	return timezone, nil
}

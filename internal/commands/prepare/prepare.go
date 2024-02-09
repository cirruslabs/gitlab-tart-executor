package prepare

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/alecthomas/units"
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/cirruslabs/gitlab-tart-executor/internal/timezone"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strconv"
	"strings"
)

var ErrFailed = errors.New("\"prepare\" stage failed")

//go:embed install-gitlab-runner-auto.sh
var installGitlabRunnerScriptAuto string

//go:embed install-gitlab-runner-brew.sh
var installGitlabRunnerBrewScript string

//go:embed install-gitlab-runner-curl.sh
var installGitlabRunnerCurlScript string

var concurrency uint64
var cpuOverrideRaw string
var memoryOverrideRaw string
var customDirectoryMounts []string
var autoPrune bool

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare a Tart VM for execution",
		RunE:  runPrepareVM,
	}

	command.PersistentFlags().Uint64Var(&concurrency, "concurrency", 1,
		"Maximum number of concurrently running Tart VMs to calculate the \"auto\" resources")
	command.PersistentFlags().StringVar(&cpuOverrideRaw, "cpu", "",
		"Override default image CPU configuration (number of CPUs or \"auto\")")
	command.PersistentFlags().StringVar(&memoryOverrideRaw, "memory", "",
		"Override default image memory configuration (size in megabytes or \"auto\")")
	command.PersistentFlags().StringArrayVar(&customDirectoryMounts, "dir", []string{},
		"\"--dir\" arguments to pass to \"tart run\", can be specified multiple times")
	command.PersistentFlags().BoolVar(&autoPrune, "auto-prune", true,
		"Whether to enable or disable the Tart's auto-pruning mechanism (sets the "+
			"TART_NO_AUTO_PRUNE environment variable for Tart command invocations under the hood)")

	return command
}

//nolint:gocognit // looks good for now
func runPrepareVM(cmd *cobra.Command, args []string) error {
	cpuOverride, err := parseCPUOverride(cmd.Context(), cpuOverrideRaw)
	if err != nil {
		return err
	}

	memoryOverride, err := parseMemoryOverride(cmd.Context(), memoryOverrideRaw)
	if err != nil {
		return err
	}

	// Auto-prune is enabled by default in Tart,
	// so we only need to act when it's set to "false"
	if !autoPrune {
		if err := os.Setenv("TART_NO_AUTO_PRUNE", "true"); err != nil {
			return err
		}
	}

	gitLabEnv, err := gitlab.InitEnv()
	if err != nil {
		return err
	}

	config, err := tart.NewConfigFromEnvironment()
	if err != nil {
		return err
	}

	if config.AlwaysPull {
		log.Printf("Pulling the latest version of %s...\n", gitLabEnv.JobImage)

		pullArgs := []string{"pull", gitLabEnv.JobImage}

		if config.InsecurePull {
			pullArgs = append(pullArgs, "--insecure")
		}

		if config.PullConcurrency != 0 {
			pullArgs = append(pullArgs, "--concurrency",
				strconv.FormatUint(uint64(config.PullConcurrency), 10))
		}

		_, _, err := tart.TartExecWithEnv(cmd.Context(), additionalPullEnv(gitLabEnv.Registry), pullArgs...)
		if err != nil {
			return err
		}
	}

	log.Println("Cloning and configuring a new VM...")
	vm, err := tart.CreateNewVM(cmd.Context(), *gitLabEnv, cpuOverride, memoryOverride)
	if err != nil {
		return err
	}
	err = vm.Start(config, gitLabEnv, customDirectoryMounts)
	if err != nil {
		return err
	}

	// Monitor "tart run" command's output so it's not silenced
	go vm.MonitorTartRunOutput()

	log.Println("Waiting for the VM to boot and be SSH-able...")
	ssh, err := vm.OpenSSH(cmd.Context(), config)
	if err != nil {
		return err
	}

	log.Println("Was able to SSH!")

	installGitlabRunnerScript, err := installGitlabRunnerScript(config.InstallGitlabRunner)
	if err != nil {
		return err
	}

	if installGitlabRunnerScript != "" {
		log.Println("Installing GitLab Runner...")

		session, err := ssh.NewSession()
		if err != nil {
			return err
		}
		defer session.Close()

		session.Stdin = bytes.NewBufferString(installGitlabRunnerScript)
		session.Stdout = os.Stdout
		session.Stderr = os.Stderr

		if err := session.Shell(); err != nil {
			return err
		}

		if err := session.Wait(); err != nil {
			return err
		}
	}

	if config.Timezone != "" {
		log.Println("Setting timezone...")

		tz, err := timezone.Parse(config.Timezone)
		if err != nil {
			return err
		}

		session, err := ssh.NewSession()
		if err != nil {
			return err
		}
		defer session.Close()

		if err := session.Run(fmt.Sprintf("sudo systemsetup settimezone %s", tz)); err != nil {
			return err
		}

		log.Printf("Timezone was set to %s!\n", tz)
	}

	log.Println("VM is ready.")

	return ssh.Close()
}

func additionalPullEnv(registry *gitlab.Registry) map[string]string {
	// Prefer manual registry credentials override from the user
	tartRegistryUsername, tartRegistryUsernameOK := os.LookupEnv("CUSTOM_ENV_TART_REGISTRY_USERNAME")
	tartRegistryPassword, tartRegistryPasswordOK := os.LookupEnv("CUSTOM_ENV_TART_REGISTRY_PASSWORD")
	if tartRegistryUsernameOK && tartRegistryPasswordOK {
		result := map[string]string{
			"TART_REGISTRY_USERNAME": tartRegistryUsername,
			"TART_REGISTRY_PASSWORD": tartRegistryPassword,
		}

		tartRegistryHostname, tartRegistryHostnameOK := os.LookupEnv("CUSTOM_ENV_TART_REGISTRY_HOSTNAME")
		if tartRegistryHostnameOK {
			result["TART_REGISTRY_HOSTNAME"] = tartRegistryHostname
		}

		return result
	}

	// Otherwise fallback to GitLab's provided registry credentials, if any
	if registry != nil {
		return map[string]string{
			"TART_REGISTRY_HOSTNAME": registry.Address,
			"TART_REGISTRY_USERNAME": registry.User,
			"TART_REGISTRY_PASSWORD": registry.Password,
		}
	}

	return nil
}

func parseCPUOverride(ctx context.Context, override string) (uint64, error) {
	// No override
	if override == "" {
		return 0, nil
	}

	// "Auto" override
	if override == "auto" {
		count, err := cpu.CountsWithContext(ctx, true)
		if err != nil {
			return 0, err
		}

		return uint64(count) / concurrency, nil
	}

	// Exact override
	return strconv.ParseUint(override, 10, 64)
}

func parseMemoryOverride(ctx context.Context, override string) (uint64, error) {
	// No override
	if override == "" {
		return 0, nil
	}

	// "Auto" override
	if override == "auto" {
		virtualMemoryStat, err := mem.VirtualMemoryWithContext(ctx)
		if err != nil {
			return 0, err
		}

		return virtualMemoryStat.Total / uint64(units.MiB) / concurrency, nil
	}

	// Exact override
	return strconv.ParseUint(override, 10, 64)
}

func installGitlabRunnerScript(installGitlabRunner string) (string, error) {
	switch installGitlabRunner {
	case "brew":
		return installGitlabRunnerBrewScript, nil
	case "curl":
		return installGitlabRunnerCurlScript, nil
	case "true", "yes", "on":
		log.Printf("%q value for TART_EXECUTOR_INSTALL_GITLAB_RUNNER will deprecated "+
			"in next version, please use either \"brew\", \"curl\" or \"major.minor.patch\"",
			installGitlabRunner)

		return installGitlabRunnerScriptAuto, nil
	case "":
		return "", nil
	default:
		version, err := semver.NewVersion(installGitlabRunner)
		if err == nil {
			return strings.ReplaceAll(installGitlabRunnerCurlScript, "${GITLAB_RUNNER_VERSION}",
				"v"+version.String()), nil
		}

		return "", fmt.Errorf("%w: TART_EXECUTOR_INSTALL_GITLAB_RUNNER only accepts "+
			"\"brew\", \"curl\" or \"major.minor.patch\", got %q", ErrFailed, installGitlabRunner)
	}
}

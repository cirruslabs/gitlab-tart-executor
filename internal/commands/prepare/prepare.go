package prepare

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/alecthomas/units"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"github.com/cirruslabs/gitlab-tart-executor/internal/localnetworkhelper"
	"github.com/cirruslabs/gitlab-tart-executor/internal/tart"
	"github.com/cirruslabs/gitlab-tart-executor/internal/timezone"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/spf13/cobra"
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
var customDiskMounts []string
var autoPrune bool
var allowedImagePatterns []string
var defaultImage string
var nested bool
var tartRunEnv []string

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare a Tart VM for execution",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runPrepareVM(cmd, args); err != nil {
				return gitlab.NewSystemFailureError(err)
			}

			return nil
		},
	}

	command.PersistentFlags().Uint64Var(&concurrency, "concurrency", 1,
		"Maximum number of concurrently running Tart VMs to calculate the \"auto\" resources")
	command.PersistentFlags().StringVar(&cpuOverrideRaw, "cpu", "",
		"Override default image CPU configuration (number of CPUs or \"auto\")")
	command.PersistentFlags().StringVar(&memoryOverrideRaw, "memory", "",
		"Override default image memory configuration (size in megabytes or \"auto\")")
	command.PersistentFlags().StringArrayVar(&customDirectoryMounts, "dir", []string{},
		"\"--dir\" arguments to pass to \"tart run\", can be specified multiple times")
	command.PersistentFlags().StringArrayVar(&customDiskMounts, "disk", []string{},
		"\"--disk\" arguments to pass to \"tart run\", can be specified multiple times")
	command.PersistentFlags().BoolVar(&autoPrune, "auto-prune", true,
		"Whether to enable or disable the Tart's auto-pruning mechanism (sets the "+
			"TART_NO_AUTO_PRUNE environment variable for Tart command invocations under the hood)")
	command.PersistentFlags().StringArrayVar(&allowedImagePatterns, "allow-image", []string{},
		"only allow running images that match the given doublestar-compatible pattern, "+
			"can be specified multiple times (e.g. --allow-image \"ghcr.io/cirruslabs/macos-sonoma-*\")")
	command.PersistentFlags().StringVar(&defaultImage, "default-image", "",
		"A fallback Tart image to use, in case the job does not specify one")
	command.PersistentFlags().BoolVar(&nested, "nested", false,
		"Run the VM with nested virtualization enabled")
	command.PersistentFlags().StringArrayVar(&tartRunEnv, "tart-run-env", []string{},
		"environment variable overrides for \"tart run\"")

	localnetworkhelper.IntroduceFlag(command)

	return command
}

//nolint:gocognit,gocyclo,nestif // looks good for now
func runPrepareVM(cmd *cobra.Command, _ []string) error {
	dialer, err := localnetworkhelper.ConnectAndDropPrivileges(cmd.Context())
	if err != nil {
		return err
	}

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

	if gitLabEnv.JobImage == "" {
		if defaultImage == "" {
			return fmt.Errorf("%w: CUSTOM_ENV_CI_JOB_ID is missing and no --default-image was set", ErrFailed)
		}

		gitLabEnv.JobImage = defaultImage
		log.Printf("No image provided, falling back to default: %s\n", defaultImage)
	}

	if err := ensureImageIsAllowed(gitLabEnv.JobImage); err != nil {
		return err
	}

	config, err := tart.NewConfigFromEnvironment()
	if err != nil {
		return err
	}

	additionalCloneAndPullEnv := additionalPullEnv(gitLabEnv.Registry)

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

		_, _, err := tart.TartExecWithEnv(cmd.Context(), additionalCloneAndPullEnv, pullArgs...)
		if err != nil {
			return err
		}
	}

	vm, err := tart.CreateNewVM(cmd.Context(), gitLabEnv.VirtualMachineID(), gitLabEnv.JobImage,
		config, cpuOverride, memoryOverride, additionalCloneAndPullEnv)
	if err != nil {
		return err
	}
	err = vm.Start(config, gitLabEnv, customDirectoryMounts, customDiskMounts, nested, tartRunEnv)
	if err != nil {
		return err
	}

	// Monitor "tart run" command's output so it's not silenced
	go vm.MonitorTartRunOutput()

	log.Println("Waiting for the VM to boot and be SSH-able...")
	ssh, err := vm.OpenSSH(cmd.Context(), config, dialer)
	if err != nil {
		return err
	}
	defer ssh.Close()

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

		stdinBuf, err := session.StdinPipe()
		if err != nil {
			return err
		}

		session.Stdout = os.Stdout
		session.Stderr = os.Stderr

		if err := session.Shell(); err != nil {
			return err
		}

		// Expose GitLab job environment variables to GitLab Runner installation script
		for _, keyAndValue := range os.Environ() {
			keyAndValue, found := strings.CutPrefix(keyAndValue, "CUSTOM_ENV_")
			if !found {
				// Doesn't look like a GitLab job environment variable
				continue
			}

			key, value, _ := strings.Cut(keyAndValue, "=")

			if _, err := fmt.Fprintf(stdinBuf, "export %s=%s\n", key, quote(value)); err != nil {
				return err
			}
		}

		// Perform GitLab Runner installation
		if _, err := stdinBuf.Write([]byte(installGitlabRunnerScript)); err != nil {
			return err
		}

		if err := stdinBuf.Close(); err != nil {
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

	type MountPoint struct {
		Name string
		Path string
	}

	vmInfo, err := vm.Info(cmd.Context())
	if err != nil {
		return err
	}

	var mountPoints []MountPoint

	if _, ok := os.LookupEnv(tart.EnvTartExecutorInternalBuildsDirOnHost); ok {
		mountPoints = append(mountPoints, MountPoint{
			Name: "buildsdir",
			Path: os.Getenv(tart.EnvTartExecutorInternalBuildsDir),
		})
	}
	if _, ok := os.LookupEnv(tart.EnvTartExecutorInternalCacheDirOnHost); ok {
		mountPoints = append(mountPoints, MountPoint{
			Name: "cachedir",
			Path: os.Getenv(tart.EnvTartExecutorInternalCacheDir),
		})
	}

	for _, mountPoint := range mountPoints {
		log.Printf("Mounting %s on %s...\n", mountPoint.Name, mountPoint.Path)

		session, err := ssh.NewSession()
		if err != nil {
			return err
		}
		defer session.Close()

		var command string

		if vmInfo.OS == "darwin" {
			command = "mount_virtiofs"
		} else {
			command = "sudo mount -t virtiofs"
		}

		mkdirScript := fmt.Sprintf("mkdir -p %s", mountPoint.Path)
		mountScript := fmt.Sprintf("%s tart.virtiofs.%s.%s %s", command, mountPoint.Name,
			gitLabEnv.JobID, mountPoint.Path)
		session.Stdin = bytes.NewBufferString(strings.Join([]string{mkdirScript, mountScript, ""}, "\n"))
		session.Stdout = os.Stdout
		session.Stderr = os.Stderr

		if err := session.Shell(); err != nil {
			return err
		}

		if err := session.Wait(); err != nil {
			return err
		}
	}

	log.Println("VM is ready.")

	return nil
}

func ensureImageIsAllowed(image string) error {
	if len(allowedImagePatterns) == 0 {
		return nil
	}

	for _, allowedImagePattern := range allowedImagePatterns {
		match, err := doublestar.Match(allowedImagePattern, image)
		if err != nil {
			return err
		}

		if match {
			return nil
		}
	}

	return fmt.Errorf("image %q is disallowed by GitLab Runner configuration", image)
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

		//nolint:gosec // there's no overflow since cpu.CountsWithContext() returns positive values
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

func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

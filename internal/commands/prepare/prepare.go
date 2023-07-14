package prepare

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
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
)

//go:embed install-gitlab-runner.sh
var installGitlabRunnerScript string

var concurrency uint64
var cpuOverrideRaw string
var memoryOverrideRaw string
var customDirectoryMounts []string

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

	return command
}

func runPrepareVM(cmd *cobra.Command, args []string) error {
	cpuOverride, err := parseCPUOverride(cmd.Context(), cpuOverrideRaw)
	if err != nil {
		return err
	}

	memoryOverride, err := parseMemoryOverride(cmd.Context(), memoryOverrideRaw)
	if err != nil {
		return err
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
		_, _, err := tart.TartExecWithEnv(cmd.Context(), additionalPullEnv(gitLabEnv.Registry),
			"pull", gitLabEnv.JobImage)
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

	// Monitor "tart run" command's output to detect errors
	go vm.MonitorOutput()

	log.Println("Waiting for the VM to boot and be SSH-able...")
	ssh, err := vm.OpenSSH(cmd.Context(), config)
	if err != nil {
		return err
	}

	log.Println("Was able to SSH!")

	if config.InstallGitlabRunner {
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

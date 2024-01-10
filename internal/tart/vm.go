package tart

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	tartCommandName         = "tart"
	tartCommandHomebrewPath = "/opt/homebrew/bin/tart"
)

var (
	ErrTartNotFound = errors.New("tart command not found")
	ErrTartFailed   = errors.New("tart command returned non-zero exit code")
	ErrVMFailed     = errors.New("VM errored")
)

type VM struct {
	id string
}

func ExistingVM(gitLabEnv gitlab.Env) *VM {
	return &VM{
		id: gitLabEnv.VirtualMachineID(),
	}
}

func CreateNewVM(
	ctx context.Context,
	gitLabEnv gitlab.Env,
	cpuOverride uint64,
	memoryOverride uint64,
) (*VM, error) {
	vm := &VM{
		id: gitLabEnv.VirtualMachineID(),
	}

	if err := vm.cloneAndConfigure(ctx, gitLabEnv, cpuOverride, memoryOverride); err != nil {
		return nil, fmt.Errorf("failed to clone the VM: %w", err)
	}

	return vm, nil
}

func (vm *VM) cloneAndConfigure(
	ctx context.Context,
	gitLabEnv gitlab.Env,
	cpuOverride uint64,
	memoryOverride uint64,
) error {
	_, _, err := TartExec(ctx, "clone", gitLabEnv.JobImage, vm.id)
	if err != nil {
		return err
	}

	if cpuOverride != 0 {
		_, _, err = TartExec(ctx, "set", "--cpu", strconv.FormatUint(cpuOverride, 10), vm.id)
		if err != nil {
			return err
		}
	}

	if memoryOverride != 0 {
		_, _, err = TartExec(ctx, "set", "--memory", strconv.FormatUint(memoryOverride, 10), vm.id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (vm *VM) Start(config Config, gitLabEnv *gitlab.Env, customDirectoryMounts []string) error {
	var runArgs = []string{"run"}

	if config.Softnet {
		runArgs = append(runArgs, "--net-softnet")
	}

	if config.Headless {
		runArgs = append(runArgs, "--no-graphics")
	}

	for _, customDirectoryMount := range customDirectoryMounts {
		runArgs = append(runArgs, "--dir", customDirectoryMount)
	}

	if config.HostDir {
		runArgs = append(runArgs, "--dir", fmt.Sprintf("hostdir:%s", gitLabEnv.HostDirPath()))
	} else if buildsDir, ok := os.LookupEnv(EnvTartExecutorInternalBuildsDir); ok {
		runArgs = append(runArgs, "--dir", fmt.Sprintf("buildsdir:%s", buildsDir))
	}

	if cacheDir, ok := os.LookupEnv(EnvTartExecutorInternalCacheDir); ok {
		runArgs = append(runArgs, "--dir", fmt.Sprintf("cachedir:%s", cacheDir))
	}

	runArgs = append(runArgs, vm.id)

	tartCommandPath, err := tartCommandPath()
	if err != nil {
		return err
	}

	cmd := exec.Command(tartCommandPath, runArgs...)

	outputFile, err := os.OpenFile(vm.tartRunOutputPath(), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	cmd.Stdout = outputFile
	cmd.Stderr = outputFile

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Process.Release()
}

func (vm *VM) MonitorTartRunOutput() {
	outputFile, err := os.Open(vm.tartRunOutputPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open VM's output file, "+
			"looks like the VM wasn't started in \"prepare\" step?\n")

		return
	}
	defer func() {
		_ = outputFile.Close()
	}()

	for {
		n, err := io.Copy(os.Stdout, outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to display VM's output: %v\n", err)

			break
		}
		if n == 0 {
			time.Sleep(100 * time.Millisecond)

			continue
		}
	}
}

func (vm *VM) OpenSSH(ctx context.Context, config Config) (*ssh.Client, error) {
	ip, err := vm.IP(ctx)
	if err != nil {
		return nil, err
	}
	addr := ip + ":22"

	var netConn net.Conn
	if err := retry.Do(func() error {
		dialer := net.Dialer{}

		netConn, err = dialer.DialContext(ctx, "tcp", addr)

		return err
	}, retry.Context(ctx)); err != nil {
		return nil, fmt.Errorf("%w: failed to connect via SSH: %v", ErrVMFailed, err)
	}

	sshConfig := &ssh.ClientConfig{
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		User: config.SSHUsername,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.SSHPassword),
		},
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to connect via SSH: %v", ErrVMFailed, err)
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}

func (vm *VM) IP(ctx context.Context) (string, error) {
	stdout, _, err := TartExec(ctx, "ip", "--wait", "60", vm.id)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout), nil
}

func (vm *VM) Stop() error {
	_, _, err := TartExec(context.Background(), "stop", vm.id)

	return err
}

func (vm *VM) Delete() error {
	_, _, err := TartExec(context.Background(), "delete", vm.id)
	if err != nil {
		return fmt.Errorf("%w: failed to delete VM %s: %v", ErrVMFailed, vm.id, err)
	}

	return nil
}

func TartExec(
	ctx context.Context,
	args ...string,
) (string, string, error) {
	return TartExecWithEnv(ctx, nil, args...)
}

func TartExecWithEnv(
	ctx context.Context,
	env map[string]string,
	args ...string,
) (string, string, error) {
	tartCommandPath, err := tartCommandPath()
	if err != nil {
		return "", "", err
	}

	cmd := exec.CommandContext(ctx, tartCommandPath, args...)

	// Base environment
	cmd.Env = cmd.Environ()

	// Environment overrides
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		var exitError *exec.ExitError

		if errors.As(err, &exitError) {
			// Tart command failed, redefine the error
			// to be the Tart-specific output
			err = fmt.Errorf("%w: %q", ErrTartFailed, firstNonEmptyLine(stderr.String(), stdout.String()))
		}
	}

	return stdout.String(), stderr.String(), err
}

func firstNonEmptyLine(outputs ...string) string {
	for _, output := range outputs {
		for _, line := range strings.Split(output, "\n") {
			if line != "" {
				return line
			}
		}
	}

	return ""
}

func (vm *VM) tartRunOutputPath() string {
	// GitLab Runner redefines the TMPDIR environment variable for
	// custom executors[1] and cleans it up (you can check that by
	// following the "cmdOpts.Dir" xrefs, so we don't need to bother
	// with that ourselves.
	//
	//nolint:lll
	// [1]: https://gitlab.com/gitlab-org/gitlab-runner/-/blob/8f29a2558bd9e72bee1df34f6651db5ba48df029/executors/custom/command/command.go#L53
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-tart-run-output.log", vm.id))
}

func tartCommandPath() (string, error) {
	result, err := exec.LookPath(tartCommandName)
	if err != nil {
		// Perhaps GitLab Runner was invoked from a launchd user agent
		// with a limited PATH[1], check if Tart is available in the
		// Homebrew's binary directory before completely failing.
		//
		// [1]: https://github.com/cirruslabs/gitlab-tart-executor/issues/47
		_, err := os.Stat(tartCommandHomebrewPath)
		if err == nil {
			return tartCommandHomebrewPath, nil
		}

		return "", fmt.Errorf("%w: %s command not found in PATH, make sure Tart is installed",
			ErrTartNotFound, tartCommandName)
	}

	return result, nil
}

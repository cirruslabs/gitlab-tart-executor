package tart

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cirruslabs/gitlab-tart-executor/internal/gitlab"
	"golang.org/x/crypto/ssh"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

const tartCommandName = "tart"
const nohupCommandName = "nohup"

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

func (vm *VM) Start(config Config, gitLabEnv *gitlab.Env) error {
	var runArgs = []string{tartCommandName, "run"}

	if config.Softnet {
		runArgs = append(runArgs, "--net-softnet")
	}

	if config.Headless {
		runArgs = append(runArgs, "--no-graphics")
	}

	if config.HostDir {
		runArgs = append(runArgs, "--dir", fmt.Sprintf("hostdir:%s", gitLabEnv.HostDirPath()))
	}

	runArgs = append(runArgs, vm.id)

	cmd := exec.Command(nohupCommandName, runArgs...)

	err := cmd.Start()
	if err != nil {
		return err
	}

	// we need to release the process and nohup ensures it will survive after prepare exits
	// run will use this VM for running scripts for stages
	return cmd.Process.Release()
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
	cmd := exec.CommandContext(ctx, tartCommandName, args...)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", "", fmt.Errorf("%w: %s command not found in PATH, make sure Tart is installed",
				ErrTartNotFound, tartCommandName)
		}

		if _, ok := err.(*exec.ExitError); ok {
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

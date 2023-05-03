# GitLab Tart Executor

Custom [GitLab Runner](https://docs.gitlab.com/runner/) executor to run jobs inside ephemeral [Tart](https://tart.run/) macOS virtual machines.

## Configuration

```bash
brew install cirruslabs/cli/gitlab-tart-executor
```

```toml
concurrent = 2

[[runners]]
  # ...
  executor = "custom"
  [runners.feature_flags]
    FF_RESOLVE_FULL_TLS_CHAIN = false
  [runners.custom]
    config_exec = "gitlab-tart-executor"
    config_args = ["config"]
    prepare_exec = "gitlab-tart-executor"
    prepare_args = ["prepare"]
    run_exec = "gitlab-tart-executor"
    run_args = ["run"]
    cleanup_exec = "gitlab-tart-executor"
    cleanup_args = ["cleanup"]
```

Now you can use Tart Images in your `.gitlab-ci.yml`:

```yaml
# You can use any remote Tart image.
# Tart Executor will pull it from the registry
# and use it for creating ephemeral VMs.
image: ghcr.io/cirruslabs/macos-ventura-base:latest

test:
  # In case you tagged runners that have
  # GitLab Tart Executor configured on them
  tags:
    - tart-installed

  script:
    - uname -a
```

## Advanced configuration

### Speeding up execution by mounting a temporary directory from the host

It's been noted that jobs run faster when they write to a volume mounted from the host (most likely because this avoids the copy-on-write expansion of the VM's disk).

Tart Executor will mount a temporary directory from the host automatically if you set the `TART_EXECUTOR_HOST_DIR` variable either in GitLab UI or in `.gitlab-ci.yml`:

```yaml
test:
  # You can use any remote Tart image.
  # Tart Executor will pull it from the registry
  # and use it for creating ephemeral VMs.
  image: ghcr.io/cirruslabs/macos-ventura-base:latest

  # In case you tagged runners that have
  # GitLab Tart Executor configured on them
  tags:
    - tart-installed

  script:
    - uname -a

  variables:
    TART_EXECUTOR_HOST_DIR: true
```

### Fully utilizing resources of the host

You can tell the Tart Executor to override the default CPU and memory settings of the VM image by passing the `--cpu` and `--memory` command-line arguments to `prepare` sub-command.

To avoid manually retrieving and calculating the total number CPUs and the memory on the host, pass the `auto` as an argument to `--cpu` and `--memory` instead of the numerical values, e.g. `--cpu auto` or `--memory auto`.

This will force the `prepare` stage to retrieve the total host resources internally and calculate them according to formula:

```
<total amount of the resource (CPUs or memory) on the host> / <concurrency>
```

...where `<concurrency>` is controlled by the [`--concurrency` command-line argument](#prepare-stage).

Here's an example on how to configure the GitLab Runner to run two Tart VMs concurrently, utilizing half of the host's resources for each VM:

```toml
concurrent = 2

[[runners]]
  # ...
  executor = "custom"
  [runners.feature_flags]
    FF_RESOLVE_FULL_TLS_CHAIN = false
  [runners.custom]
    config_exec = "gitlab-tart-executor"
    config_args = ["config"]
    prepare_exec = "gitlab-tart-executor"
    prepare_args = ["prepare", "--concurrency 2", "--cpu auto", "--memory auto"]
    run_exec = "gitlab-tart-executor"
    run_args = ["run"]
    cleanup_exec = "gitlab-tart-executor"
    cleanup_args = ["cleanup"]
```

### Using different SSH credentials

Tart Executor uses the default `admin:admin` credentials when connecting to the VM over SSH.

If your image uses different credentials, set `TART_EXECUTOR_SSH_USERNAME` and/or `TART_EXECUTOR_SSH_PASSWORD` variables either in GitLab UI or in `.gitlab-ci.yml`:

```yaml
test:
  # You can use any remote Tart image.
  # Tart Executor will pull it from the registry
  # and use it for creating ephemeral VMs.
  image: ghcr.io/cirruslabs/macos-ventura-base:latest

  # In case you tagged runners that have
  # GitLab Tart Executor configured on them
  tags:
    - tart-installed

  script:
    - uname -a

  variables:
    TART_EXECUTOR_SSH_USERNAME: "custom-username"
    TART_EXECUTOR_SSH_PASSWORD: "custom-password"
```

## Licensing

Tart Executor is open sourced under MIT license so people can base their own executors in Go of this code.
Tart itself on the other hand is [source available under Fair Software License](https://tart.run/licensing/)
that required paid sponsorship upon exceeding a free limit.

## Supported command-line arguments

### `prepare` stage

| Argument        | Default     | Description                                                                       |
|-----------------|-------------|-----------------------------------------------------------------------------------|
| `--concurrency` | 1           | Maximum number of concurrently running Tart VMs to calculate the `auto` resources |
| `--cpu`         | no override | Override default image CPU configuration (number of CPUs or `auto`<sup>1</sup>)            |
| `--memory`      | no override | Override default image memory configuration (size in megabytes or `auto`<sup>1</sup>)      |

<sup>1</sup>: automatically distributes all host resources according to the concurrency level (for example, VM gets all of the host CPU and RAM assigned when `--concurrency` is 1, and half of that when `--concurrency` is 2)

## Supported environment variables

| Name                                 | Default        | Description                                                                                                                          |
|--------------------------------------|----------------|--------------------------------------------------------------------------------------------------------------------------------------|
| `TART_EXECUTOR_SSH_USERNAME`         | admin          | SSH username to use when connecting to the VM                                                                                        |
| `TART_EXECUTOR_SSH_PASSWORD`         | admin          | SSH password to use when connecting to the VM                                                                                        |
| `TART_EXECUTOR_HEADLESS`             | true           | Run the VM in headless mode (`true`) or with GUI (`false`)                                                                           |
| `TART_EXECUTOR_ALWAYS_PULL`          | true           | Always pull the latest version of the Tart image (`true`) or only when the image doesn't exist locally (`false`)                     |
| `TART_EXECUTOR_SOFTNET`              | false          | Whether to enable [Softnet](https://github.com/cirruslabs/softnet) software networking (`true`) or disable it (`false`)              |
| `TART_EXECUTOR_HOST_DIR`<sup>1</sup> | false          | Whether to mount a temporary directory from the host for performance reasons (`true`) or use a directory inside of a guest (`false`) |
| `TART_EXECUTOR_SHELL`                | system default | Alternative [Unix shell](https://en.wikipedia.org/wiki/Unix_shell) to use (e.g. `zsh` or `bash`)                                     |

<sup>1</sup>: to use the directory mounting feature, both the host and the guest need to run macOS 13.0 (Ventura) or newer.

## Local Development

In order to test a local change with your GitLab Runner, you first need to build the binary:

```bash
go build -o gitlab-tart-executor cmd/gitlab-tart-executor/main.go
```

Now you can run your GitLab Runner as follows:

```
PATH=$PATH:$PWD gitlab-runner run
```

If that's not possible, use an absolute path to `gitlab-tart-executor` binary in your `.gitlab-runner/config.toml` for `config_exec `, `prepare_exec`, `run_exec` and `cleanup_exec` fields.

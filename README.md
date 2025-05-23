# GitLab Tart Executor

Custom [GitLab Runner](https://docs.gitlab.com/runner/) executor to run jobs inside ephemeral [Tart](https://tart.run/) macOS virtual machines.

> [!IMPORTANT]
>
> **macOS 15 (Sequoia)**
>
> In case you've upgraded and encountering an issue below:
>
> ```
> Waiting for the VM to boot and be SSH-able...
> ```
>
> This is likely related to the [newly introduced "Local Network" permission](https://developer.apple.com/documentation/technotes/tn3179-understanding-local-network-privacy) on macOS Sequoia and the fact that GitLab Runner's binary might have no `LC_UUID` identifier, which is critical for the local network privacy mechanism.
>
> Make sure you have installed the latest GitLab Runner (`>=17.6.0`) [from Homebrew](https://formulae.brew.sh/formula/gitlab-runner).
>
> Homebrew version [includes a fix for lacking `LC_UUID`](https://github.com/Homebrew/homebrew-core/commit/77fcd447733f6f063ef4f635202d3748fdfb8e26) and it should ask you for a "Local Network" permission correctly when GitLab Tart Executor tries to establish connection with the Tart VMs.

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
    TART_EXECUTOR_HOST_DIR: "true"
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
    prepare_args = ["prepare", "--concurrency", "2", "--cpu", "auto", "--memory", "auto"]
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

### `config` stage

| Argument                         | Default | Description                                                                                                                                                                                           |
|----------------------------------|---------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `--builds-dir`                   |         | Path to a directory on host to use for storing builds, automatically mounts that directory to the guest VM (mutually exclusive with `--guest-builds-dir`)                                             |
| `--cache-dir`                    |         | Path to a directory on host to use for caching purposes, automatically mounts that directory to the guest VM (mutually exclusive with `--guest-cache-dir`)                                            |
| `--guest-builds-dir`<sup>1</sup> |         | Path to a directory in guest to use for storing builds, useful when mounting a block device (via [`--disk` command-line argument](#prepare-stage)) to the VM (mutually exclusive with `--builds-dir`) |
| `--guest-cache-dir`<sup>1</sup>  |         | Path to a directory in guest to use for caching purposes, useful when mounting a block device (via [`--disk` command-line argument](#prepare-stage) to the VM (mutually exclusive with `--cache-dir`) |

<sup>1</sup>: this is an advanced feature which should only be resorted to when the standard directory sharing via `--builds-dir` and `--cache-dir` is not sufficient for some reason.

### `prepare` stage

| Argument          | Default     | Description                                                                                                                                                     |
|-------------------|-------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `--concurrency`   | 1           | Maximum number of concurrently running Tart VMs to calculate the `auto` resources                                                                               |
| `--cpu`           | no override | Override default image CPU configuration (number of CPUs or `auto`<sup>1</sup>)                                                                                 |
| `--memory`        | no override | Override default image memory configuration (size in megabytes or `auto`<sup>1</sup>)                                                                           |
| `--dir`           |             | `--dir` arguments to pass to `tart run`, can be specified multiple times                                                                                        |
| `--disk`          |             | `--disk` arguments to pass to `tart run`, can be specified multiple times                                                                                       |
| `--auto-prune`    | true        | Whether to enable or disable the Tart's auto-pruning mechanism (sets the `TART_NO_AUTO_PRUNE` environment variable for Tart command invocations under the hood) |
| `--allow-image`   |             | only allow running images that match the given [doublestar](https://github.com/bmatcuk/doublestar)-compatible pattern, can be specified multiple times          |
| `--default-image` |             | A fallback Tart image to use, in case the job does not specify one                                                                                              |
| `--nested`        | false       | Run VMs with [nested virtualization](https://tart.run/faq/#nested-virtualization-support) enabled                                                            |

<sup>1</sup>: automatically distributes all host resources according to the concurrency level (for example, VM gets all of the host CPU and RAM assigned when `--concurrency` is 1, and half of that when `--concurrency` is 2)

## Supported environment variables

| Name                                  | Default        | Description                                                                                                                                                                                                                                                                                                                                                                                                                              |
|---------------------------------------|----------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `TART_EXECUTOR_ALWAYS_PULL`           | true           | Always pull the latest version of the Tart image (`true`) or only when the image doesn't exist locally (`false`)                                                                                                                                                                                                                                                                                                                         |
| `TART_EXECUTOR_BRIDGED`               |                | Use bridged networking, for example, "en0". Use `tart run --net-bridged=list` to see names of all available interfaces.                                                                                                                                                                                                                                                                                                                  |
| `TART_EXECUTOR_HEADLESS`              | true           | Run the VM in headless mode (`true`) or with GUI (`false`)                                                                                                                                                                                                                                                                                                                                                                               |
| `TART_EXECUTOR_HOST_DIR`<sup>1</sup>  | false          | Whether to mount a temporary directory from the host for performance reasons (`true`) or use a directory inside of a guest (`false`)                                                                                                                                                                                                                                                                                                     |
| `TART_EXECUTOR_INSECURE_PULL`         | false          | Set to `true` to connect the OCI registry via insecure HTTP protocol                                                                                                                                                                                                                                                                                                                                                                     |
| `TART_EXECUTOR_INSTALL_GITLAB_RUNNER` |                | Set to `brew` to install GitLab Runner [via Homebrew](https://docs.gitlab.com/runner/install/osx.html#homebrew-installation-alternative), `curl` to install the latest version [using cURL](https://docs.gitlab.com/runner/install/osx.html#manual-installation-official) or `major.minor.patch` to install a specific version [using cURL](https://docs.gitlab.com/runner/install/bleeding-edge.html#download-any-other-tagged-release) |
| `TART_EXECUTOR_PULL_CONCURRENCY`      |                | Override the Tart's default network concurrency parameter (`--concurrency`) when pulling remote VMs from the OCI-compatible registries                                                                                                                                                                                                                                                                                                   |
| `TART_EXECUTOR_RANDOM_MAC`            | true           | Generate a new MAC address and therefore use a unique local IP address for every cloned VM                                                                                                                                                                                                                                                                                                                                               |
| `TART_EXECUTOR_ROOT_DISK_OPTS`        |                | When set, this value will be passed to `tart run`'s `--root-disk-opts` command-line argument.                                                                                                                                                                                                                                                                                                                                            |
| `TART_EXECUTOR_SHELL`                 | system default | Alternative [Unix shell](https://en.wikipedia.org/wiki/Unix_shell) to use (e.g. `bash -l`)                                                                                                                                                                                                                                                                                                                                               |
| `TART_EXECUTOR_SOFTNET_ALLOW`         |                | Comma-separated list of CIDRs to allow the traffic to when using Softnet isolation                                                                                                                                                                                                                                                                                                                                                       |
| `TART_EXECUTOR_SOFTNET`               | false          | Whether to enable [Softnet](https://github.com/cirruslabs/softnet) software networking (`true`) or disable it (`false`)                                                                                                                                                                                                                                                                                                                  |
| `TART_EXECUTOR_SSH_PASSWORD`          | admin          | SSH password to use when connecting to the VM                                                                                                                                                                                                                                                                                                                                                                                            |
| `TART_EXECUTOR_SSH_PORT`              | 22             | Connect to the VM at the given SSH port                                                                                                                                                                                                                                                                                                                                                                                                  |
| `TART_EXECUTOR_SSH_USERNAME`          | admin          | SSH username to use when connecting to the VM                                                                                                                                                                                                                                                                                                                                                                                            |
| `TART_EXECUTOR_TIMEZONE`              |                | Timezone to set in the guest (or `auto` to pick up the timezone from host), see `systemsetup listtimezones` for a list of possible timezones                                                                                                                                                                                                                                                                                             |

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

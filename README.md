Custom GitLab Runner Executor to run jobs inside ephemeral [Tart](https://tart.run/) macOS virtual machines.

# Configuration

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
# You can use any remote Tart Image.
# Tart Executor will pull it from the registry and use it for creating ephemeral VMs.
image: ghcr.io/cirruslabs/macos-ventura-base:latest

test:
  tags:
    - tart-installed # in case you tagged runners with Tart Executor installed
  script:
    - uname -a
```

# Licensing

Tart Executor is open sourced under MIT license so people can base their own executors in Go of this code.
Tart itself on the other hand is [source available under Fair Software License](https://tart.run/licensing/)
that required paid sponsorship upon exceeding a free limit.

## Supported command-line arguments

### `prepare` stage

| Argument  | Default           | Description                                                     |
|-----------|-------------------|-----------------------------------------------------------------|
| `--cpu`   | `0` (no override) | Override default image CPU configuration (number of CPUs)       |
| `--memory` | `0` (no override) | Override default image memory configuration (size in megabytes) |

## Supported environment variables

| Name                                 | Default | Description                                                                                                                          |
|--------------------------------------|---------|--------------------------------------------------------------------------------------------------------------------------------------|
| `TART_EXECUTOR_SSH_USERNAME`         | admin   | SSH username to use when connecting to the VM                                                                                        |
| `TART_EXECUTOR_SSH_PASSWORD`         | admin   | SSH password to use when connecting to the VM                                                                                        |
| `TART_EXECUTOR_HEADLESS`             | true    | Run the VM in headless mode (`true`) or with GUI (`false`)                                                                           |
| `TART_EXECUTOR_ALWAYS_PULL`          | true    | Always pull the latest version of the Tart image (`true`) or only when the image doesn't exist locally (`false`)                     |
| `TART_EXECUTOR_SOFTNET`              | false   | Whether to enable [Softnet](https://github.com/cirruslabs/softnet) software networking (`true`) or disable it (`false`)              |
| `TART_EXECUTOR_HOST_DIR`<sup>1</sup> | false   | Whether to mount a temporary directory from the host for performance reasons (`true`) or use a directory inside of a guest (`false`) |

<sup>1</sup>: to use the directory mounting feature, both the host and the guest need to run macOS 13.0 (Ventura) or newer.

# Local Development

In order to test a local change with your GitLab Runner, you first need to build the binary:

```bash
go build -o gitlab-tart-executor cmd/gitlab-tart-executor/main.go
```

Now you can run your GitLab Runner as follows:

```
PATH=$PATH:$PWD gitlab-runner run
```

If that's not possible, use an absolute path to `gitlab-tart-executor` binary in your `.gitlab-runner/config.toml` for `config_exec `, `prepare_exec`, `run_exec` and `cleanup_exec` fields.

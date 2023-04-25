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
  builds_dir = "/Users/admin/builds"
  cache_dir = "/Users/admin/cache"
  [runners.feature_flags]
    FF_RESOLVE_FULL_TLS_CHAIN = false
  [runners.custom]
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

## Supported environment variables

| Name                      | Default | Description                                                                                                             |
|---------------------------|--------------|-------------------------------------------------------------------------------------------------------------------------|
| `CIRRUS_GTE_SSH_USERNAME` | admin        | SSH username to use when connecting to the VM                                                                           |
| `CIRRUS_GTE_SSH_PASSWORD` | admin        | SSH password to use when connecting to the VM                                                                           |
| `CIRRUS_GTE_HEADLESS`     | true         | Run the VM in headless mode (`true`) or with GUI (`false`)                                                              |
| `CIRRUS_GTE_ALWAYS_PULL`  | true         | Always pull the latest version of the Tart image (`true`) or only when the image doesn't exist locally (`false`)        |
| `CIRRUS_GTE_SOFTNET`      | false        | Whether to enable [Softnet](https://github.com/cirruslabs/softnet) software networking (`true`) or disable it (`false`) |
| `CIRRUS_GTE_CPU`          |              | Override default image CPU configuration, e.g. `8` (number of CPUs)                                                     |
| `CIRRUS_GTE_MEMORY`       |              | Override default image memory configuration, e.g. `8192` (size in megabytes)                                                               |

# Local Development

In order to test a local change with your GitLab Runner, you first need to build the binary:

```bash
go build -o executor-dev cmd/gitlab-tart-executor/main.go
```

Now you can use an absolute path to `executor-dev` binary in your `.gitlab-runner/config.toml` configuration
in `prepare_exec`, `run_exec` and `cleanup_exec` fields.

#!/bin/bash

# Set shell options to enable fail-fast behavior
#
# * -e: fail the script when an error occurs or command fails
# * -u: fail the script when attempting to reference unset parameters
# * -o pipefail: by default an exit status of a pipeline is that of its
#                last command, this fails the pipe early if an error in
#                any of its commands occurs
#
set -euo pipefail

function install_via_brew() {
  if test "$1" != "latest"; then
    echo "Installing specific GitLab Runner version via Homebrew is not supported"

    exit 1
  fi

  echo "Installing GitLab Runner via Homebrew..."
  brew install gitlab-runner
}

function install_via_curl() {
  echo "Installing GitLab Runner using cURL..."

  if ! sudo -n true &>/dev/null
  then
    echo "Failed to install GitLab Runner using cURL: passwordless sudo is required, but is not configured"

    exit 1
  fi

  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)
  case $ARCH in
    aarch64) ARCH="arm64" ;;
    x86_64)  ARCH="amd64" ;;
  esac

  GITLAB_RUNNER_URL="https://{{ .GitlabRunnerProvider }}/${1}/binaries/gitlab-runner-${OS}-${ARCH}"
  GITLAB_RUNNER_FILE=$(mktemp "${TMPDIR:-/tmp}/gitlab-runner.XXXXXX")
  GITLAB_RUNNER_PATH="/usr/local/bin/gitlab-runner"

  echo "Downloading GitLab Runner from $GITLAB_RUNNER_URL..."
  curl --output "$GITLAB_RUNNER_FILE" "$GITLAB_RUNNER_URL"
  chmod +x "$GITLAB_RUNNER_FILE"

  # /usr/local is empty on fresh macOS installations
  # (for example, try ghcr.io/cirruslabs/macos-ventura-vanilla:latest)
  if test ! -e /usr/local/bin
  then
    echo "Creating /usr/local/bin because it doesn't exist yet..."
    sudo mkdir -p /usr/local/bin
  fi

  echo "Installing $GITLAB_RUNNER_PATH executable..."
  sudo mv -v "$GITLAB_RUNNER_FILE" "$GITLAB_RUNNER_PATH"
}

# Is GitLab Runner already installed?
if type gitlab-runner &> /dev/null
then
  echo "GitLab Runner is already installed, skipping installation"

  exit 0
fi

{{ range .Environ -}}
export "{{ . }}"
{{ end }}

{{- if eq .PackageManager "brew" }}
install_via_brew "{{ .GitlabRunnerVersion }}"
{{- else if eq .PackageManager "curl" }}
install_via_curl "{{ .GitlabRunnerVersion }}"
{{- else }}
if type brew &> /dev/null
then
  install_via_brew "{{ .GitlabRunnerVersion }}"
elif type curl &> /dev/null
then
  install_via_curl "{{ .GitlabRunnerVersion }}"
else
  echo "Failed to install GitLab Runner: neither Homebrew nor cURL are available!"

  exit 1
fi
{{- end }}

echo "GitLab Runner was successfully installed!"

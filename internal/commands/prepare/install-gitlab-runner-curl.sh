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

GITLAB_RUNNER_URL="https://gitlab-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-runner-darwin-arm64"
GITLAB_RUNNER_PATH="/usr/local/bin/gitlab-runner"

# Is GitLab Runner already installed?
if type gitlab-runner &> /dev/null
then
  echo "GitLab Runner is already installed, skipping installation"

  exit 0
fi

echo "Installing GitLab Runner using cURL..."

if ! sudo -n true &>/dev/null
then
  echo "Failed to install GitLab Runner using cURL: passwordless sudo is required, but is not configured"

  exit 1
fi

# /usr/local is empty on fresh macOS installations
# (for example, try ghcr.io/cirruslabs/macos-ventura-vanilla:latest)
if test ! -e /usr/local/bin
then
  echo "Creating /usr/local/bin because it doesn't exist yet..."
  sudo mkdir -p /usr/local/bin
fi

echo "Downloading GitLab Runner from $GITLAB_RUNNER_URL to $GITLAB_RUNNER_PATH..."
sudo curl --output "$GITLAB_RUNNER_PATH" "$GITLAB_RUNNER_URL"

echo "Making $GITLAB_RUNNER_PATH executable..."
sudo chmod +x "$GITLAB_RUNNER_PATH"

echo "GitLab Runner was successfully installed!"

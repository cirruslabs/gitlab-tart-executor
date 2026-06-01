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

# Is GitLab Runner already installed?
if [ "${TART_EXECUTOR_UPGRADE_GITLAB_RUNNER:-}" != "true" ] && type gitlab-runner &> /dev/null
then
  echo "GitLab Runner is already installed, skipping installation"

  exit 0
fi

echo "Updating Homebrew..."

brew update

echo "Installing GitLab Runner via Homebrew..."

if type gitlab-runner &> /dev/null
then
  brew reinstall gitlab-runner
else
  brew install gitlab-runner
fi

echo "GitLab Runner was successfully installed!"

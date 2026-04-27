#!/usr/bin/env sh
# Install docker-container-healthchecker as a Docker CLI plugin in the
# current user's ~/.docker/cli-plugins/ directory.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/dokku/docker-container-healthchecker/main/install.sh | sh
#   VERSION=v0.14.1 curl -fsSL https://raw.githubusercontent.com/dokku/docker-container-healthchecker/main/install.sh | sh
#
# Environment variables:
#   VERSION    Release tag to install (defaults to the latest release; requires v0.10.0 or later).
#   PLUGIN_DIR Override destination directory (defaults to $HOME/.docker/cli-plugins).

set -eu

NAME="docker-container-healthchecker"
PLUGIN_NAME="docker-healthcheck"
REPO="dokku/docker-container-healthchecker"
PLUGIN_DIR="${PLUGIN_DIR:-${HOME}/.docker/cli-plugins}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  linux | darwin) ;;
  *)
    echo "error: unsupported OS: $os" >&2
    exit 1
    ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  aarch64 | arm64) arch="arm64" ;;
  *)
    echo "error: unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

if [ -z "${VERSION:-}" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name": "\(.*\)".*/\1/p' | head -n1)"
fi

if [ -z "$VERSION" ]; then
  echo "error: could not determine latest version; set VERSION explicitly" >&2
  exit 1
fi

# Release assets are raw binaries named like docker-container-healthchecker-<os>-<arch>.
asset="${NAME}-${os}-${arch}"
url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

echo "downloading ${url}"
curl -fsSL "$url" -o "${tmpdir}/${asset}"

mkdir -p "$PLUGIN_DIR"
install -m 0755 "${tmpdir}/${asset}" "${PLUGIN_DIR}/${PLUGIN_NAME}"

echo "installed ${NAME} ${VERSION} to ${PLUGIN_DIR}/${PLUGIN_NAME}"
echo "try: docker healthcheck version"

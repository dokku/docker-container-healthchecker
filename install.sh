#!/usr/bin/env sh
# Install docker-container-healthchecker as a Docker CLI plugin in the
# current user's ~/.docker/cli-plugins/ directory.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/dokku/docker-container-healthchecker/main/install.sh | sh
#   VERSION=v0.11.3 curl -fsSL https://raw.githubusercontent.com/dokku/docker-container-healthchecker/main/install.sh | sh
#
# Environment variables:
#   VERSION    Release tag to install (defaults to the latest release).
#   PLUGIN_DIR Override destination directory (defaults to $HOME/.docker/cli-plugins).

set -eu

NAME="docker-container-healthchecker"
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

# Release assets are tarballs named like docker-container-healthchecker_<VER>_<os>_<arch>.tgz
# containing a single binary docker-container-healthchecker-<arch>.
asset="${NAME}_${VERSION#v}_${os}_${arch}.tgz"
url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

echo "downloading ${url}"
curl -fsSL "$url" -o "${tmpdir}/${asset}"
tar -xzf "${tmpdir}/${asset}" -C "$tmpdir"

mkdir -p "$PLUGIN_DIR"
install -m 0755 "${tmpdir}/${NAME}-${arch}" "${PLUGIN_DIR}/${NAME}"

echo "installed ${NAME} ${VERSION} to ${PLUGIN_DIR}/${NAME}"
echo "try: docker container-healthchecker version"

# Getting Started

`docker-container-healthchecker` is a Docker CLI plugin that validates whether a Docker container is healthy by running configurable checks -- HTTP requests, port listening, command execution, or uptime monitoring -- defined in an `app.json` file.

## Why docker-container-healthchecker?

After starting a container, you need to know it is actually working before routing traffic to it. Docker's built-in `HEALTHCHECK` instruction supports a single command, but many applications need more: checking an HTTP endpoint, verifying a port is bound, or running an internal validation script.

Container orchestrators like Kubernetes distinguish between startup, liveness, and readiness probes, each serving a different purpose. Docker alone does not make this distinction.

docker-container-healthchecker fills this gap. You define multiple healthchecks per process type in an `app.json` file, run them against a live container, and get pass/fail results. It supports all three probe types and four check strategies, giving you Kubernetes-style health validation for plain Docker containers.

## Installation

Once installed, the plugin is available as `docker healthcheck`:

```bash
docker healthcheck version
```

### Quick Install (Linux and macOS)

Use the install script to download the latest release and install it as a Docker CLI plugin:

```bash
curl -fsSL https://raw.githubusercontent.com/dokku/docker-container-healthchecker/main/install.sh | sh
```

To install a specific version:

```bash
VERSION=v0.11.3 curl -fsSL https://raw.githubusercontent.com/dokku/docker-container-healthchecker/main/install.sh | sh
```

To install to a custom directory:

```bash
PLUGIN_DIR=/usr/libexec/docker/cli-plugins curl -fsSL https://raw.githubusercontent.com/dokku/docker-container-healthchecker/main/install.sh | sh
```

### Homebrew (macOS)

```bash
brew install dokku/repo/docker-container-healthchecker
```

### Debian/Ubuntu

```bash
sudo apt-get update
sudo apt-get install docker-container-healthchecker
```

The Debian package installs the binary to both `/usr/bin/docker-container-healthchecker` (for direct invocation) and `/usr/libexec/docker/cli-plugins/docker-healthcheck` (for automatic Docker CLI plugin discovery).

### Binary Download

Download a pre-built binary from [GitHub Releases](https://github.com/dokku/docker-container-healthchecker/releases) and place it in your Docker CLI plugins directory:

```bash
mkdir -p ~/.docker/cli-plugins
cp docker-healthcheck ~/.docker/cli-plugins/docker-healthcheck
chmod +x ~/.docker/cli-plugins/docker-healthcheck
```

Docker looks for plugins in these directories:

- `~/.docker/cli-plugins/` (per-user)
- `/usr/libexec/docker/cli-plugins/` (system-wide on Linux)
- `/usr/local/lib/docker/cli-plugins/` (system-wide on Linux)
- `$(brew --prefix)/lib/docker/cli-plugins/` (Homebrew on macOS)

### From Source

Build and install as a Docker CLI plugin:

```bash
make install
```

This builds the binary for your platform and copies it to `~/.docker/cli-plugins/docker-healthcheck`.

> **Direct invocation:** The binary can also be run directly as `docker-container-healthchecker <command>` without installing it as a plugin.

## Your First Healthcheck

Start a container that serves HTTP traffic:

```bash
docker run -d --name demo nginx
```

Create an `app.json` file with a path check against the root URL:

```json
{
  "healthchecks": {
    "web": [
      {
        "type": "startup",
        "name": "http check",
        "path": "/",
        "port": 80,
        "attempts": 3
      }
    ]
  }
}
```

Run the healthcheck:

```bash
docker healthcheck check demo --port 80
```

The command inspects the container, finds the healthcheck definitions for the `web` process type, sends an HTTP request to `/` on port 80, and reports the result. If the response returns a 2xx status code, the check passes.

If you run the check without an `app.json` file or without any checks for the specified process type, a default 10-second uptime check runs automatically -- verifying the container has been running without restarts.

Clean up:

```bash
docker rm -f demo
```

## What to Read Next

- [Command Reference](command-reference.md) -- all CLI flags and arguments
- [Healthchecks](healthchecks.md) -- the four check strategies and three healthcheck types
- [File Format](file-format.md) -- full `app.json` field reference
- [Dokku Migration](dokku-migration.md) -- converting from the legacy CHECKS file format

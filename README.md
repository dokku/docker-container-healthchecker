# docker-container-healthchecker

Runs healthchecks against local docker containers.

## Installation

Install as a Docker CLI plugin with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/dokku/docker-container-healthchecker/main/install.sh | sh
```

Or via Homebrew:

```bash
brew install dokku/repo/docker-container-healthchecker
```

Or build from source:

```bash
make install
```

See the [Getting Started](docs/getting-started.md#installation) guide for all distribution channels (Debian/Ubuntu packages, binary downloads, etc.).

Once installed, the plugin is available via `docker healthcheck`.

## Usage

Check a running container against healthchecks defined in `app.json`:

```bash
docker healthcheck check my-container
```

Add a default uptime healthcheck to a process type:

```bash
docker healthcheck add web
```

Add a listening check on a specific port:

```bash
docker healthcheck add web --listening-check --port 3000
```

Check if healthchecks are defined for a process type:

```bash
docker healthcheck exists web
```

Convert a legacy Dokku CHECKS file to app.json format:

```bash
docker healthcheck convert path/to/CHECKS --pretty
```

See the [command reference](docs/command-reference.md) for all flags and options.

## Documentation

- [Getting Started](docs/getting-started.md) -- why docker-container-healthchecker, installation, and your first healthcheck
- [Command Reference](docs/command-reference.md) -- all CLI flags and arguments
- [File Format](docs/file-format.md) -- the app.json healthcheck schema
- [Healthchecks](docs/healthchecks.md) -- check strategies, healthcheck types, and scheduler support
- [Dokku Migration](docs/dokku-migration.md) -- converting from the legacy CHECKS file format

## License

[MIT](LICENSE)

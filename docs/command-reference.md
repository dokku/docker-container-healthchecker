# Command Reference

## Synopsis

```bash
# As a Docker CLI plugin:
docker healthcheck <command> [arguments] [flags]

# Direct invocation:
docker-container-healthchecker <command> [arguments] [flags]
```

## check

Runs healthchecks against a running container. Reads healthcheck definitions from `app.json`, finds checks matching the specified process type and healthcheck type, and executes them in parallel. The exit code equals the number of failed checks (0 means all passed).

If no healthchecks are defined for the requested process type, a default 10-second uptime check runs automatically.

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `container-id` | Yes | ID or name of the container to check. |

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--app-json` | string | `app.json` | Path to the app.json file containing healthcheck definitions. |
| `--header` | string (repeatable) | `[]` | HTTP header in `curl -H` format for path checks. Repeat for multiple headers. |
| `--ip-address` | string | `""` | IP address override for HTTP path checks. When empty, the container IP is fetched from the Docker network. |
| `--network` | string | `bridge` | Docker network to use when fetching the container IP for path checks. |
| `--port` | int | `5000` | Default port for checks. Overridden by the `port` field in the healthcheck definition. |
| `--process-type` | string | `web` | Process type to run checks for. |
| `--type` | string | `startup` | Healthcheck type to run: `startup`, `liveness`, or `readiness`. |

### Examples

Check a container using the defaults (web process, startup type):

```bash
docker healthcheck check my-container
```

Check with a custom port and process type:

```bash
docker healthcheck check my-container --process-type worker --port 8080
```

Check with custom HTTP headers:

```bash
docker healthcheck check my-container --header 'X-Forwarded-Proto: https' --header 'Host: myapp.example.com'
```

Run liveness checks instead of startup:

```bash
docker healthcheck check my-container --type liveness
```

Use a specific network and app.json path:

```bash
docker healthcheck check my-container --network custom-net --app-json /path/to/app.json
```

## add

Adds a healthcheck entry to an `app.json` file for the specified process type. By default, it adds an uptime check. Use `--listening-check` to add a listening check instead.

Output is written to stdout unless `--in-place` is specified.

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `process-type` | No (default: `web`) | Process type to add the check to. |

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--app-json` | string | `app.json` | Path to the app.json file to update. If the file does not exist, an empty app.json is assumed. |
| `--if-empty` | bool | `false` | Only add the check if no healthchecks exist for the process type. |
| `--in-place` | bool | `false` | Write the result back to the app.json file instead of stdout. |
| `--listening-check` | bool | `false` | Add a listening check instead of an uptime check. |
| `--name` | string | `""` | Name for the healthcheck. Defaults to `default` when not specified. |
| `--port` | int | `5000` | Port for the listening check. |
| `--pretty` | bool | `false` | Pretty-print the JSON output. |
| `--type` | string | `startup` | Healthcheck type: `startup`, `liveness`, or `readiness`. |
| `--uptime` | int | `1` | Minimum uptime in seconds for uptime checks. |
| `--warn-only` | bool | `false` | Mark the check as warn-only so failures produce warnings but do not fail the service. |

### Examples

Add a default uptime check for the web process:

```bash
docker healthcheck add web
```

Add a listening check on port 3000:

```bash
docker healthcheck add web --listening-check --port 3000
```

Add only if no checks exist yet:

```bash
docker healthcheck add web --if-empty
```

Write directly to the app.json file with pretty printing:

```bash
docker healthcheck add web --in-place --pretty
```

Add a check for a worker process:

```bash
docker healthcheck add worker --uptime 30
```

## exists

Checks whether the `app.json` file contains any healthchecks for the specified process type. Produces no output on success. Prints an error message and exits non-zero if no healthchecks are found.

This is useful in scripts to conditionally run healthchecks only when they are defined.

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `process-type` | Yes | Process type to check for healthchecks. |

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--app-json` | string | `app.json` | Path to the app.json file. |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Healthchecks exist for the process type. |
| `1` | No healthchecks found, or the file could not be read. |

### Examples

Check if the web process has healthchecks:

```bash
docker healthcheck exists web
```

Check with a custom app.json path:

```bash
docker healthcheck exists web --app-json /path/to/app.json
```

Use in a script:

```bash
if docker healthcheck exists web; then
  docker healthcheck check my-container
fi
```

## convert

Converts a legacy Dokku `CHECKS` file into the `app.json` healthcheck format. All converted checks are assigned to the `web` process type with the `startup` healthcheck type.

See the [Dokku Migration](dokku-migration.md) guide for details on the CHECKS file format and what changes during conversion.

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `check-file` | Yes | Path to the CHECKS file to convert. |

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--app-json` | string | `""` | Path to an existing app.json file to merge into. If empty, starts from a blank file. |
| `--in-place` | bool | `false` | Write the result back to the app.json file instead of stdout. Requires `--app-json`. |
| `--pretty` | bool | `false` | Pretty-print the JSON output. |

### Examples

Convert to stdout:

```bash
docker healthcheck convert CHECKS
```

Convert with pretty printing:

```bash
docker healthcheck convert CHECKS --pretty
```

Merge into an existing app.json:

```bash
docker healthcheck convert CHECKS --app-json app.json --pretty
```

Convert and write in place:

```bash
docker healthcheck convert CHECKS --app-json app.json --in-place --pretty
```

## See Also

- [Healthchecks](healthchecks.md) -- how check strategies and healthcheck types work
- [File Format](file-format.md) -- full app.json field reference
- [Dokku Migration](dokku-migration.md) -- CHECKS file format and conversion details

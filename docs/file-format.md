# File Format

Healthchecks are defined in an `app.json` file, grouped by process type. Each process type maps to an array of healthcheck objects, so you can run multiple checks against a single container -- for example, an HTTP path check and a command check together.

Unknown keys in the healthcheck object are ignored, so you can add fields like `description` for your own documentation without affecting behavior.

## Structure

The top-level key is `healthchecks`, containing an object where each key is a process type (e.g., `web`, `worker`) and each value is an array of healthcheck objects:

```json
{
  "healthchecks": {
    "web": [
      {
        "type": "startup",
        "name": "web check",
        "path": "/health/ready",
        "attempts": 3
      }
    ],
    "worker": [
      {
        "type": "startup",
        "name": "worker uptime",
        "uptime": 10
      }
    ]
  }
}
```

## Fields

Each healthcheck object supports the following fields:

| Field | Default | Description | Scheduler aliases |
|-------|---------|-------------|-------------------|
| `attempts` | `3` | Number of retry attempts on failure. | `nomad=check_restart.limit` |
| `command` | `[]` | Command to execute inside the container as a JSON array of strings. | `kubernetes=exec.Command` `nomad=command args` |
| `content` | `""` | String to search for in HTTP response body. Only used with `path` checks. | |
| `httpHeaders` | `[]` | List of headers to add to HTTP requests. Each entry has `name` and `value` fields. | `kubernetes=httpHeaders` |
| `initialDelay` | `0` (seconds) | Seconds to wait after container start before running the check. Gives the application time to initialize. | `kubernetes=initialDelaySeconds` `nomad=check_restart.grace` |
| `listening` | `false` | When `true`, performs a listening check instead of the default uptime check. | |
| `name` | auto-generated | Human-readable name for the healthcheck. If omitted, a name is generated from the healthcheck definition. | `nomad=name` |
| `onFailure` | `null` | Action to take when the healthcheck fails. See [Failure hooks](#failure-hooks). | |
| `path` | `/` (for HTTP checks) | HTTP path to request. Setting this field activates a path check. | `kubernetes=httpGet.path` `nomad=path` |
| `port` | `5000` | Port to run the healthcheck against. Can be overridden by the `--port` CLI flag. | `kubernetes=port` |
| `scheme` | `http` | URL scheme for HTTP checks. Must be `http` or `https`. | `kubernetes=scheme` |
| `timeout` | `5` (seconds) | Seconds to wait before a single healthcheck attempt times out. | `kubernetes=timeoutSeconds` `nomad=timeout` |
| `type` | `""` | Purpose of the healthcheck: `startup`, `liveness`, or `readiness`. See [Healthchecks](healthchecks.md#healthcheck-types). | |
| `uptime` | `0` (seconds) | Minimum seconds the container must be running without restarting. Setting this field activates an uptime check. | |
| `wait` | `5` (seconds) | Seconds to wait between retry attempts. | `kubernetes=periodSeconds` `nomad=interval` |
| `warn` | `false` | When `true`, failures produce a warning but do not count against the service. The check result is logged but ignored for pass/fail decisions. | |

## Failure Hooks

The `onFailure` field allows you to trigger an action when a healthcheck fails. It is an object with two optional sub-fields:

| Field | Description |
|-------|-------------|
| `command` | A command to execute on the host machine as a JSON array of strings. Runs after the healthcheck fails all retry attempts. |
| `url` | A URL to send an HTTP POST request to with a JSON payload containing the healthcheck name and errors. |

Example:

```json
{
  "type": "startup",
  "path": "/health/ready",
  "onFailure": {
    "command": ["/usr/local/bin/notify-failure.sh"],
    "url": "https://hooks.example.com/healthcheck-failed"
  }
}
```

Both fields are optional -- you can use either or both. The command runs first, then the URL is posted to.

## Process Types

Healthchecks are grouped by process type. When running `docker healthcheck check`, the `--process-type` flag selects which group to execute (default: `web`). Similarly, the `add` command takes a process-type argument to determine where new checks are added.

A single process type can have multiple healthchecks. They run in parallel -- all must pass for the container to be considered healthy.

If no healthchecks are defined for the requested process type, a default 10-second uptime check runs automatically.

## Example

A complete `app.json` with multiple check strategies and healthcheck types:

```json
{
  "healthchecks": {
    "web": [
      {
        "type": "startup",
        "name": "web http check",
        "path": "/health/ready",
        "port": 8080,
        "attempts": 3,
        "timeout": 5,
        "wait": 5
      },
      {
        "type": "liveness",
        "name": "web liveness",
        "path": "/health/live",
        "port": 8080,
        "timeout": 1,
        "attempts": 5
      },
      {
        "type": "readiness",
        "name": "web command check",
        "command": ["/app/check-ready.sh"],
        "initialDelay": 10,
        "timeout": 5,
        "attempts": 5,
        "wait": 10
      }
    ],
    "worker": [
      {
        "type": "startup",
        "name": "worker uptime",
        "uptime": 30
      }
    ]
  }
}
```

## See Also

- [Healthchecks](healthchecks.md) -- how check strategies and healthcheck types work
- [Command Reference](command-reference.md) -- CLI flags that override file-format defaults (`--port`, `--type`)

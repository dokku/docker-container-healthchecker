# Healthchecks

docker-container-healthchecker validates running containers by executing one or more checks defined in an `app.json` file. Each healthcheck has two dimensions: a **check strategy** (how the check runs) and a **healthcheck type** (when and why it runs).

## Check Strategies

Each healthcheck uses exactly one check strategy, determined by which fields are set in the healthcheck definition. The four strategies are mutually exclusive -- setting `command` prevents you from also setting `path`, `uptime`, or `listening` on the same healthcheck entry.

### uptime

The simplest strategy. Verifies that the container has been running continuously for at least `uptime` seconds without restarting. This does not probe the application at all -- it only checks container-level state via the Docker API.

Use an uptime check when you want a basic smoke test or when the container has no network endpoint to probe.

```json
{
  "type": "startup",
  "name": "basic uptime",
  "uptime": 10
}
```

> The `uptime` strategy does **not** respect the `attempts`, `timeout`, or `wait` fields. If the container has restarted at any point, the check fails immediately.

### listening

Checks whether a process inside the container is listening on all network interfaces (`0.0.0.0` or `::`) for the specified port. This uses `nsenter` and `netstat` to inspect the container's network namespace from the host.

Use a listening check when you need to confirm the application has bound to a port and is ready for external connections -- for example, before a reverse proxy starts sending traffic.

```json
{
  "type": "startup",
  "name": "port check",
  "listening": true,
  "port": 3000
}
```

> The `listening` strategy respects `attempts` and `wait` but does **not** respect `timeout`.

### path

Sends an HTTP request to the container at the specified `path` and checks for a successful response (2xx status code). The container's IP address is fetched from the Docker network (default: `bridge`), and the port defaults to `5000`.

Use a path check when your application exposes an HTTP health endpoint. This is the most common strategy for web services because it validates that the application can actually serve requests, not just that the process is running.

```json
{
  "type": "startup",
  "name": "http health",
  "path": "/health/ready",
  "port": 8080,
  "timeout": 5,
  "attempts": 3
}
```

You can add custom HTTP headers to path checks. Headers can be set in the `app.json` file using the `httpHeaders` field:

```json
{
  "type": "startup",
  "path": "/health/ready",
  "httpHeaders": [
    {"name": "X-Forwarded-Proto", "value": "https"},
    {"name": "Host", "value": "myapp.example.com"}
  ]
}
```

Or via the `--header` CLI flag:

```bash
docker healthcheck check my-container --header 'X-Forwarded-Proto: https'
```

The `content` field lets you search the response body for a specific string, failing the check if the string is not found. The `scheme` field controls whether the request uses `http` or `https`.

> The `path` strategy respects `attempts`, `timeout`, and `wait`.

### command

Runs a command inside the container using `docker exec`. If the command exits with a non-zero status code, the check fails.

Use a command check when you need to validate something that cannot be checked from outside the container -- for example, running an internal script that uses environment variables only available inside the container.

```json
{
  "type": "readiness",
  "name": "internal check",
  "command": ["/app/check-ready.sh"],
  "timeout": 5,
  "attempts": 3
}
```

A command check can perform complex validations. For example, a script that curls an internal endpoint and checks for expected content:

```bash
#!/usr/bin/env bash

OUTPUT="$(curl http://localhost:$PORT/some-file.js)"
if ! grep jQuery <<< "$OUTPUT"; then
  echo "Expected in output: jQuery"
  echo "Output: $OUTPUT"
  exit 1
fi
```

> The `command` strategy respects `attempts`, `timeout`, and `wait`.

## Healthcheck Types

The `type` field specifies the purpose of a healthcheck -- when and why it runs. The three types are modeled after Kubernetes probe terminology, making it straightforward to map checks to Kubernetes deployments.

### startup

Determines whether a container has started successfully and is ready to accept traffic. This is the default type and the most commonly used. The container is not considered running until all startup checks pass.

Use startup checks to gate deployments -- if the check fails, the container is treated as a failed deployment.

### liveness

Determines whether a running container is still healthy. A liveness failure indicates the container is in a broken state (e.g., deadlocked or stuck) and should be restarted.

Use liveness checks for long-running services where the process might still be running but is no longer functioning correctly.

### readiness

Determines whether a running container is ready to accept new requests. Unlike liveness, a readiness failure does not trigger a restart -- it temporarily removes the container from receiving traffic until it recovers.

Use readiness checks for services that may become temporarily overloaded or need time to warm caches.

## Scheduler Support

Not all container schedulers support every healthcheck type. The following table shows which types are supported by each scheduler:

| Scheduler | `startup` | `liveness` | `readiness` |
|-----------|-----------|------------|-------------|
| docker-local | yes | no | no |
| k3s | yes | yes | yes |

When targeting a scheduler that does not support a given type, healthchecks of that type are ignored during deployment.

## See Also

- [File Format](file-format.md) -- all healthcheck fields including `attempts`, `timeout`, `wait`, and `initialDelay`
- [Command Reference](command-reference.md) -- flags that control check behavior (`--port`, `--header`, `--network`, `--type`)

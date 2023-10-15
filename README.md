# docker-container-healthchecker

Runs healthchecks against local docker containers

## Requirements

- golang 1.21+

## Usage

### add command

Add a healthcheck to an existing `app.json` file, specified by the `--app-json` flag. If the file does not exist, an empty `app.json` file will be assumed.

```shell
# creates a default startup uptime healthcheck
# docker-container-healthchecker add $PROCESS_TYPE
docker-container-healthchecker add web
```

By default, the output is written to `stdout`, though it can be written to the file specified via the `--in-place` flag.

```shell
docker-container-healthchecker add web --in-place
```

The `add` command supports adding a listening check, which optionally supports a `--port` flag (default: `5000`):

```shell
# listening check
docker-container-healthchecker add web --listening-check --port 3000
```

### check command

After creating an app.json file, execute the healthchecker like so:

```shell
# docker-container-healthchecker check $CONTAINER_ID_OR_NAME
docker-container-healthchecker check cb0ce984f2aa
```

By default, the checks specified for the `web` process type are executed at the `startup` type level. If the process-type has no checks specified, a default `uptime` container check of 10 seconds is performed.

### convert command

The convert command can be used to convert the `CHECKS` file format used by Dokku into the healthcheck format used by `docker-container-healthchecker`

```shell
docker-container-healthchecker convert path/to/CHECKS
```

By default, the output will be written to stdout. This output can be pretty printed using the `--pretty` flag:

```shell
docker-container-healthchecker convert path/to/CHECKS --pretty
```

The `app.json` in the current working directory is used as input. A different `app.json` file can also be updated by specifying the path to an `app.json` file via the `--app-json` flag. If the file does not exist, an error will be raised.

```shell
docker-container-healthchecker convert path/to/CHECKS --app.json path/to/app.json
```

The `app.json` file can also be modified in place instead of writing to stdout by specifying the `--in-place` flag. This also respects pretty printing via the `--pretty` flag.

```shell
docker-container-healthchecker convert path/to/CHECKS --pretty --inline
```

### exists command

Check if the `app.json` contains healthchecks for the specified process type:

```shell
docker-container-healthchecker exists web
```

If there are healthchecks, there will be no output and the exit code will be 0. An error message will be displayed and the exit code will be non-zero in all other cases.

The `app.json` in the current working directory is used as input. A different `app.json` file can also be checked by specifying the path to an `app.json` file via the `--app-json` flag. If the file does not exist, an error will be raised.

```shell
docker-container-healthchecker exists web --app.json path/to/app.json
```

### Check types

#### `command`

Runs a command within the specified container. If the command exits non-zero, the output is printed and the check is considered failed.

As the `command` type is run within the container environment, it can be used to perform dynamic checks using environment variables exposed to the container. For example, it may be used to simulate content checks on http endpoints like so:

```shell
#!/usr/bin/env bash

OUTPUT="$(curl http://localhost:$PORT/some-file.js)"
if ! grep jQuery <<< "$OUTPUT"; then
  echo "Expected in output: jQuery"
  echo "Output: $output"
  exit 1
fi
```

`command` checks respect the `attempts`, `timeout` and `wait` properties.

If the `command` type is in use, the `path` and `uptime` healthcheck properties must be empty.

#### `listening`

Checks to see if there is a container process listening on all interfaces for the specified `port`. This can be used to ensure external proxy implementations can connect to the underlying process.

`listening` checks respect the `attempts` and `wait` properties but _does not_ respect the `timeout` property.

#### `path`

Executes an http request against the container at the specified `path`. The container IP address is fetched from the `bridge` network and the port is default to `5000`, though both settings can be overridden by the `--network` and `--port` flags, respectively.

HTTP `path` checks respect the `attempts`, `timeout` and `wait` properties.

Custom headers may be specified for http `path` requests by utilizing the `--header` flag like so:

```shell
docker-container-healthchecker check cb0ce984f2aa --header 'X-Forwarded-Proto: https'
```

To further customize the type of request performed, please see the `command` check type.

If the `path` type is in use, the `command` and `uptime` healthcheck properties must be empty.

#### `uptime`

Ensures the container is up for at least `uptime` in seconds. If a container has restarted at all during that time, it is treated as an unhealthy container.

`uptime` checks _do not_ respect the `attempts`, `timeout` and `wait` properties.

If the `uptime` type is in use, the `command` and `path` healthcheck properties must be empty.

### File Format

Healthchecks are defined within a json file and have the following properties (the respective scheduler properties are also noted for comparison):

| field        | default                           | description                                     | scheduler aliases (kubernetes, nomad) |
|--------------|-----------------------------------|-------------------------------------------------|---------------------------------------|
| attempts     | default: `3`, unit: seconds       | Number of retry attempts to perform on failure. | `nomad=check_restart.limit` |
| command      | default: `""` (empty string)      | Command to execute within container.            | `kubernetes=exec.Command` `nomad=command args` |
| content      | default: `""` (empty string)      | Content to search in http path check output.    | |
| httpHeaders  | default: `[]` (for http checks)   | A list of headers (defined by `name`/`value` attributes) to add to a request. | `kubernetes=httpHeaders` |
| initialDelay | default: `0`, unit: seconds       | Number of seconds to wait after a container has started before triggering the healthcheck. | `kubernetes=initialDelaySeconds` `nomad=check_restart.grace` |
| listening    | default: `false`                  | Whether to perform a listening check or not. | |
| name         | default: `""` (autogenerated)     | The name of the healthcheck. If unspecified, it will be autogenerated from the rest of the healthcheck information. | `nomad=name` |
| scheme       | default: `http` (for http checks) | An http path to check. | `kubernetes=scheme` |
| path         | default: `/` (for http checks)    | An http path to check. | `kubernetes=httpGet.path` `nomad=path` |
| port         | default: `5000`, unit: int        | Port to run healthcheck against. Overrides flag. | `kubernetes=port` |
| timeout      | default: `5`, unit: seconds       | Number of seconds to wait before timing out a healthcheck. | `kubernetes=timeoutSeconds` `nomad=timeout` |
| type         | default: `""` (none)              | Type of the healthcheck. Options: `liveness`, `readiness`, `startup` | |
| uptime       | default: `""` (none)              | Amount of time the container must be alive before the container is considered healthy. Any restarts will cause this to check to fail, and this check does not respect retries. | |
| wait         | default: `5`, unit: seconds       | Number of seconds to wait between healthcheck attempts. | `kubernetes=periodSeconds` `nomad=interval` |
| warn         | default: `false`                  | Outputs a warning for the check but does not count failures against the service. | |

> Any extra properties are ignored

Healthchecks are specified within an `app.json` file grouped in a per process-type basis. One process type can have one or more healthchecks of various types.

```json
{
  "healthchecks": {
    "web": [
        {
            "type":        "startup",
            "name":        "web check",
            "description": "Checking if the app responds to the /health/ready endpoint",
            "path":        "/health/ready",
            "attempts": 3
        }
    ]
}
```

An example `app.json` is located in the `tests/fixtures`. Unknown keys are ignored, as in the above case with the `description` field.

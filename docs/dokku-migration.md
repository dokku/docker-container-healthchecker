# Dokku Migration

docker-container-healthchecker includes a `convert` command to migrate from the legacy `CHECKS` file format used by [Dokku](https://dokku.com) to the `app.json` healthcheck format. This is useful when transitioning an existing Dokku application to use the newer healthcheck system.

## CHECKS File Format

The `CHECKS` file is a line-based format where each line is either a variable assignment or a path definition. Blank lines and lines starting with `#` are ignored.

### Variables

Variables are set with `NAME=VALUE` syntax and apply to all checks defined after them in the file:

| Variable | Description |
|----------|-------------|
| `WAIT` | Seconds to wait between retry attempts. Maps to the `wait` field. |
| `TIMEOUT` | Seconds before a check times out. Maps to the `timeout` field. |
| `ATTEMPTS` | Number of retry attempts. Maps to the `attempts` field. |

### Path definitions

Each non-variable line defines an HTTP path check. A line can optionally include a content string to search for in the response body, separated by a space:

```
/path
/path Expected content
```

Paths can include a hostname and scheme:

```
https://example.com/path
http://example.com/path Expected content
```

When a hostname is provided, it is converted to a `Host` HTTP header in the resulting healthcheck.

### Example CHECKS file

```
WAIT=2
TIMEOUT=5
ATTEMPTS=2

/ Hello World!
```

This defines a single HTTP path check against `/` that expects the response body to contain "Hello World!", with 2 attempts, a 5-second timeout, and a 2-second wait between retries.

## Converting

Run the `convert` command with the path to your CHECKS file:

```bash
docker healthcheck convert path/to/CHECKS
```

This outputs the equivalent `app.json` content to stdout. Use `--pretty` for readable output:

```bash
docker healthcheck convert path/to/CHECKS --pretty
```

To merge the converted checks into an existing `app.json`:

```bash
docker healthcheck convert path/to/CHECKS --app-json app.json --pretty
```

To modify the `app.json` file directly instead of printing to stdout:

```bash
docker healthcheck convert path/to/CHECKS --app-json app.json --in-place --pretty
```

## What Changes

All converted checks are assigned to the `web` process type with the `startup` healthcheck type. The conversion maps fields as follows:

| CHECKS | app.json |
|--------|----------|
| `WAIT=N` | `"wait": N` |
| `TIMEOUT=N` | `"timeout": N` |
| `ATTEMPTS=N` | `"attempts": N` |
| `/path` | `"path": "/path"` |
| `Content text` | `"content": "Content text"` |
| `https://host/path` | `"scheme": "https"`, `"path": "/path"`, `"httpHeaders": [{"name": "Host", "value": "host"}]` |

Each check is automatically named `check-1`, `check-2`, etc.

## Example

Given this CHECKS file:

```
WAIT=2
TIMEOUT=5
ATTEMPTS=2

/ Hello World!
```

Running `docker healthcheck convert CHECKS --pretty` produces:

```json
{
  "healthchecks": {
    "web": [
      {
        "attempts": 2,
        "content": "Hello World!",
        "name": "check-1",
        "path": "/",
        "timeout": 5,
        "type": "startup",
        "wait": 2
      }
    ]
  }
}
```

## See Also

- [Command Reference](command-reference.md#convert) -- all `convert` command flags
- [File Format](file-format.md) -- full `app.json` field reference

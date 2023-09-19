#!/usr/bin/env bats

export SYSTEM_NAME="$(uname -s | tr '[:upper:]' '[:lower:]')"
export BIN_NAME="build/$SYSTEM_NAME/docker-container-healthchecker-amd64"

setup_file() {
  make prebuild $BIN_NAME
  rm -f app.json >/dev/null || true
  docker rm -f dch-test-1 >/dev/null || true
  docker container run -d --platform linux/amd64 --name dch-test-1 dokku/test-app:1 /start web
}

teardown_file() {
  docker rm -f dch-test-1 >/dev/null || true
  rm -f app.json >/dev/null || true
  make clean
}

setup() {
  rm -f app.json >/dev/null || true
}

teardown() {
  rm -f app.json >/dev/null || true
}

@test "[add] default" {
  run "$BIN_NAME" add web
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"name":"default","type":"startup","uptime":1}]}}'

  run "$BIN_NAME" add
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"name":"default","type":"startup","uptime":1}]}}'
}

@test "[add] default in-place" {
  run "$BIN_NAME" add web --in-place
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output_not_exists

  run cat app.json
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"name":"default","type":"startup","uptime":1}]}}'
}

@test "[add] default in-place existing" {
  run "$BIN_NAME" add web --in-place
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output_not_exists

  run cat app.json
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"name":"default","type":"startup","uptime":1}]}}'

  run "$BIN_NAME" add web --if-empty --in-place
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output_not_exists

  run cat app.json
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"name":"default","type":"startup","uptime":1}]}}'

  run "$BIN_NAME" add web --in-place --uptime 2
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output_not_exists

  run cat app.json
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"name":"default","type":"startup","uptime":1},{"name":"default","type":"startup","uptime":2}]}}'

  run "$BIN_NAME" add web --if-empty --in-place --uptime 2
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output

  run cat app.json
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"name":"default","type":"startup","uptime":1},{"name":"default","type":"startup","uptime":2}]}}'
}

@test "[add] custom uptime" {
  run "$BIN_NAME" add --uptime 10
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"name":"default","type":"startup","uptime":10}]}}'
}

@test "[add] custom process-type" {
  run "$BIN_NAME" add worker
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"worker":[{"name":"default","type":"startup","uptime":1}]}}'

  run "$BIN_NAME" add worker --uptime 10
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"worker":[{"name":"default","type":"startup","uptime":10}]}}'
}

@test "[add] existing" {
  echo '{"healthchecks":{"web":[{"name":"existing uptime check","type":"startup","uptime":5}]}}' >app.json

  run "$BIN_NAME" add --uptime 10
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"name":"existing uptime check","type":"startup","uptime":5},{"name":"default","type":"startup","uptime":10}]}}'
}

@test "[add] existing if-empty" {
  echo '{"healthchecks":{"web":[{"name":"existing uptime check","type":"startup","uptime":5}]}}' >app.json
  run "$BIN_NAME" add --if-empty --uptime 10
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"name":"existing uptime check","type":"startup","uptime":5}]}}'
}

@test "[check] uptime check" {
  echo '{"healthchecks":{"web":[{"name":"uptime check","type":"startup","uptime":5}]}}' >app.json

  run "$BIN_NAME" check dch-test-1
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output_contains "Healthcheck succeeded name='uptime check'"
  assert_output_contains "Running healthcheck name='uptime check' type='uptime' uptime=5"
}

@test "[check] path check" {
  echo '{"healthchecks":{"web":[{"name":"path check","type":"startup","path":"/"}]}}' >app.json

  run "$BIN_NAME" check dch-test-1
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output_contains "Healthcheck succeeded name='path check'"
  assert_output_contains "Running healthcheck name='path check' delay=0 path='/' retries=2 timeout=5 type='path'"
}

@test "[check] path check delay" {
  echo '{"healthchecks":{"web":[{"name":"path check","type":"startup","path":"/", "initialDelay": 10}]}}' >app.json

  run "$BIN_NAME" check dch-test-1
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output_contains "Healthcheck succeeded name='path check'"
  assert_output_contains "Running healthcheck name='path check' delay=10 path='/' retries=2 timeout=5 type='path'"
}

@test "[check] path check ip-address" {
  echo '{"healthchecks":{"web":[{"name":"path check","type":"startup","path":"/"}]}}' >app.json

  IP_ADDRESS="$(docker container inspect --format '{{range $v := .NetworkSettings.Networks}}{{printf "%s" $v.IPAddress}}{{end}}' dch-test-1)"

  run "$BIN_NAME" check dch-test-1 --ip-address "$IP_ADDRESS"
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output_contains "Healthcheck succeeded name='path check'"
  assert_output_contains "Running healthcheck name='path check' delay=0 path='/' retries=2 timeout=5 type='path'"
}

@test "[check] command check" {
  echo '{"healthchecks":{"web":[{"command":["echo","hi"],"name":"command check","type":"startup"}]}}' >app.json

  run "$BIN_NAME" check dch-test-1
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output_contains "Healthcheck succeeded name='command check'"
  assert_output_contains "Running healthcheck name='command check' attempts=3 command='[echo hi]' timeout=5 type='command' wait=5"
}

@test "[convert] checks-root" {
  run "$BIN_NAME" convert tests/fixtures/checks-root.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"attempts":2,"name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] hostname" {
  run "$BIN_NAME" convert tests/fixtures/hostname.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"attempts":2,"content":"nodejs/express","httpHeaders":[{"name":"Host","value":"example.com"}],"name":"check-1","path":"/path","timeout":5,"type":"startup","wait":2}]}}'
}

@test "[convert] hostname-scheme" {
  run "$BIN_NAME" convert tests/fixtures/hostname-scheme.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"attempts":2,"content":"nodejs/express","httpHeaders":[{"name":"Host","value":"example.com"}],"name":"check-1","path":"/path","scheme":"https","timeout":5,"type":"startup","wait":2}]}}'
}

@test "[convert] dockerfile-app-json-formations" {
  run "$BIN_NAME" convert tests/fixtures/dockerfile-app-json-formations.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"attempts":2,"content":"nodejs/express","name":"check-1","path":"/","timeout":5,"type":"startup","wait":2}]}}'
}

@test "[convert] dockerfile-noexpose" {
  run "$BIN_NAME" convert tests/fixtures/dockerfile-noexpose.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"attempts":2,"content":"Hello World!","name":"check-1","path":"/","timeout":5,"type":"startup","wait":2}]}}'
}

@test "[convert] dockerfile-procfile-bad" {
  run "$BIN_NAME" convert tests/fixtures/dockerfile-procfile-bad.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"attempts":2,"content":"nodejs/express","name":"check-1","path":"/","timeout":5,"type":"startup","wait":2}]}}'
}

@test "[convert] dockerfile-procfile" {
  run "$BIN_NAME" convert tests/fixtures/dockerfile-procfile.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"attempts":2,"content":"nodejs/express","name":"check-1","path":"/","timeout":5,"type":"startup","wait":2}]}}'
}

@test "[convert] dockerfile" {
  run "$BIN_NAME" convert tests/fixtures/dockerfile.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"attempts":2,"content":"Hello World!","name":"check-1","path":"/","timeout":5,"type":"startup","wait":2}]}}'
}

@test "[convert] gitsubmodules" {
  run "$BIN_NAME" convert tests/fixtures/gitsubmodules.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"Hello","name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] go-fail-postdeploy" {
  run "$BIN_NAME" convert tests/fixtures/go-fail-postdeploy.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"go","name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] go-fail-predeploy" {
  run "$BIN_NAME" convert tests/fixtures/go-fail-predeploy.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"go","name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] go" {
  run "$BIN_NAME" convert tests/fixtures/go.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"go","name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] java" {
  run "$BIN_NAME" convert tests/fixtures/java.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"Hello from Java","name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] multi" {
  run "$BIN_NAME" convert tests/fixtures/multi.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"Heroku Multi Buildpack on Dokku","name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] nodejs-express-noappjson" {
  run "$BIN_NAME" convert tests/fixtures/nodejs-express-noappjson.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"attempts":2,"content":"nodejs/express","name":"check-1","path":"/","timeout":5,"type":"startup","wait":2}]}}'
}

@test "[convert] nodejs-express-noprocfile" {
  run "$BIN_NAME" convert tests/fixtures/nodejs-express-noprocfile.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"nodejs/express","name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] php" {
  run "$BIN_NAME" convert tests/fixtures/php.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"\u003chtml\u003e\u003ch3\u003ephp\u003c/h3\u003e\u003c/html\u003e","name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] python-flask" {
  run "$BIN_NAME" convert tests/fixtures/python-flask.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"python/flask","name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] python" {
  run "$BIN_NAME" convert tests/fixtures/python.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"attempts":2,"content":"python/http.server","name":"check-1","path":"/","timeout":7,"type":"startup","wait":2}]}}'
}

@test "[convert] ruby" {
  run "$BIN_NAME" convert tests/fixtures/ruby.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"Hello, world","name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] static" {
  run "$BIN_NAME" convert tests/fixtures/static.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"Static Page","name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] zombies" {
  run "$BIN_NAME" convert tests/fixtures/zombies-buildpack.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"go","name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] zombies-dockerfile-no-tini" {
  run "$BIN_NAME" convert tests/fixtures/zombies-dockerfile-no-tini.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"go","name":"check-1","path":"/","type":"startup"}]}}'
}

@test "[convert] zombies-dockerfile-tini" {
  run "$BIN_NAME" convert tests/fixtures/zombies-dockerfile-tini.CHECKS
  echo "output: $output"
  echo "status: $status"
  assert_success
  assert_output '{"healthchecks":{"web":[{"content":"go","name":"check-1","path":"/","type":"startup"}]}}'
}

flunk() {
  {
    if [[ "$#" -eq 0 ]]; then
      cat -
    else
      echo "$*"
    fi
  }
  return 1
}

assert_equal() {
  if [[ "$1" != "$2" ]]; then
    {
      echo "expected: $1"
      echo "actual:   $2"
    } | flunk
  fi
}

assert_failure() {
  if [[ "$status" -eq 0 ]]; then
    flunk "expected failed exit status"
  elif [[ "$#" -gt 0 ]]; then
    assert_output "$1"
  fi
}

assert_success() {
  if [[ "$status" -ne 0 ]]; then
    flunk "command failed with exit status $status"
  elif [[ "$#" -gt 0 ]]; then
    assert_output "$1"
  fi
}

assert_output() {
  local expected
  if [[ $# -eq 0 ]]; then
    expected="$(cat -)"
  else
    expected="$1"
  fi
  assert_equal "$expected" "$output"
}

assert_output_contains() {
  local input="$output"
  local expected="$1"
  local count="${2:-1}"
  local found=0
  until [ "${input/$expected/}" = "$input" ]; do
    input="${input/$expected/}"
    found=$((found + 1))
  done
  assert_equal "$count" "$found"
}

assert_output_not_exists() {
  [[ -z "$output" ]] || flunk "expected no output, found some"
}

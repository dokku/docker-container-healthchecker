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

@test "[check] uptime check" {
  echo '{"healthchecks":{"web":[{"name":"uptime check","type":"startup","uptime":5}]}}' >app.json

  run "$BIN_NAME" check dch-test-1
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" =~ "Healthcheck succeeded name='uptime check'" ]]
  [[ "$output" =~ "Running healthcheck name='uptime check' type='uptime' uptime=5" ]]
}

@test "[check] path check" {
  echo '{"healthchecks":{"web":[{"name":"path check","type":"startup","path":"/"}]}}' >app.json

  run "$BIN_NAME" check dch-test-1
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" =~ "Healthcheck succeeded name='path check'" ]]
  [[ "$output" =~ "Running healthcheck name='path check' delay=0 path='/' retries=2 timeout=5 type='path'" ]]
}

@test "[check] command check" {
  echo '{"healthchecks":{"web":[{"command":["echo","hi"],"name":"command check","type":"startup"}]}}' >app.json

  run "$BIN_NAME" check dch-test-1
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" =~ "Healthcheck succeeded name='command check'" ]]
  [[ "$output" =~ "Running healthcheck name='command check' attempts=3 command='[echo hi]' timeout=5 type='command' wait=5" ]]
}

@test "[add] default" {
  run "$BIN_NAME" add web
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"name":"default","type":"startup","uptime":1}]}}' ]]

  run "$BIN_NAME" add
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"name":"default","type":"startup","uptime":1}]}}' ]]
}

@test "[add] custom uptime" {
  run "$BIN_NAME" add --uptime 10
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"name":"default","type":"startup","uptime":10}]}}' ]]
}

@test "[add] custom process-type" {
  run "$BIN_NAME" add worker
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"worker":[{"name":"default","type":"startup","uptime":1}]}}' ]]

  run "$BIN_NAME" add worker --uptime 10
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"worker":[{"name":"default","type":"startup","uptime":10}]}}' ]]
}

@test "[convert] checks-root" {
  run "$BIN_NAME" convert tests/fixtures/checks-root.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"attempts":2,"path":"/"}]}}' ]]
}

@test "[convert] hostname" {
  run "$BIN_NAME" convert tests/fixtures/hostname.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"attempts":2,"content":"nodejs/express","httpHeaders":[{"name":"Host","value":"example.com"}],"path":"/path","timeout":5,"wait":2}]}}' ]]
}

@test "[convert] hostname-scheme" {
  run "$BIN_NAME" convert tests/fixtures/hostname-scheme.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"attempts":2,"content":"nodejs/express","httpHeaders":[{"name":"Host","value":"example.com"}],"path":"/path","scheme":"https","timeout":5,"wait":2}]}}' ]]
}

@test "[convert] dockerfile-app-json-formations" {
  run "$BIN_NAME" convert tests/fixtures/dockerfile-app-json-formations.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"attempts":2,"content":"nodejs/express","path":"/","timeout":5,"wait":2}]}}' ]]
}

@test "[convert] dockerfile-noexpose" {
  run "$BIN_NAME" convert tests/fixtures/dockerfile-noexpose.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"attempts":2,"content":"Hello World!","path":"/","timeout":5,"wait":2}]}}' ]]
}

@test "[convert] dockerfile-procfile-bad" {
  run "$BIN_NAME" convert tests/fixtures/dockerfile-procfile-bad.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"attempts":2,"content":"nodejs/express","path":"/","timeout":5,"wait":2}]}}' ]]
}

@test "[convert] dockerfile-procfile" {
  run "$BIN_NAME" convert tests/fixtures/dockerfile-procfile.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"attempts":2,"content":"nodejs/express","path":"/","timeout":5,"wait":2}]}}' ]]
}

@test "[convert] dockerfile" {
  run "$BIN_NAME" convert tests/fixtures/dockerfile.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"attempts":2,"content":"Hello World!","path":"/","timeout":5,"wait":2}]}}' ]]
}

@test "[convert] gitsubmodules" {
  run "$BIN_NAME" convert tests/fixtures/gitsubmodules.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"Hello","path":"/"}]}}' ]]
}

@test "[convert] go-fail-postdeploy" {
  run "$BIN_NAME" convert tests/fixtures/go-fail-postdeploy.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"go","path":"/"}]}}' ]]
}

@test "[convert] go-fail-predeploy" {
  run "$BIN_NAME" convert tests/fixtures/go-fail-predeploy.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"go","path":"/"}]}}' ]]
}

@test "[convert] go" {
  run "$BIN_NAME" convert tests/fixtures/go.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"go","path":"/"}]}}' ]]
}

@test "[convert] java" {
  run "$BIN_NAME" convert tests/fixtures/java.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"Hello from Java","path":"/"}]}}' ]]
}

@test "[convert] multi" {
  run "$BIN_NAME" convert tests/fixtures/multi.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"Heroku Multi Buildpack on Dokku","path":"/"}]}}' ]]
}

@test "[convert] nodejs-express-noappjson" {
  run "$BIN_NAME" convert tests/fixtures/nodejs-express-noappjson.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"attempts":2,"content":"nodejs/express","path":"/","timeout":5,"wait":2}]}}' ]]
}

@test "[convert] nodejs-express-noprocfile" {
  run "$BIN_NAME" convert tests/fixtures/nodejs-express-noprocfile.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"nodejs/express","path":"/"}]}}' ]]
}

@test "[convert] php" {
  run "$BIN_NAME" convert tests/fixtures/php.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"\u003chtml\u003e\u003ch3\u003ephp\u003c/h3\u003e\u003c/html\u003e","path":"/"}]}}' ]]
}

@test "[convert] python-flask" {
  run "$BIN_NAME" convert tests/fixtures/python-flask.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"python/flask","path":"/"}]}}' ]]
}

@test "[convert] python" {
  run "$BIN_NAME" convert tests/fixtures/python.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"attempts":2,"content":"python/http.server","path":"/","timeout":7,"wait":2}]}}' ]]
}

@test "[convert] ruby" {
  run "$BIN_NAME" convert tests/fixtures/ruby.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"Hello, world","path":"/"}]}}' ]]
}

@test "[convert] static" {
  run "$BIN_NAME" convert tests/fixtures/static.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"Static Page","path":"/"}]}}' ]]
}

@test "[convert] zombies" {
  run "$BIN_NAME" convert tests/fixtures/zombies-buildpack.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"go","path":"/"}]}}' ]]
}

@test "[convert] zombies-dockerfile-no-tini" {
  run "$BIN_NAME" convert tests/fixtures/zombies-dockerfile-no-tini.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"go","path":"/"}]}}' ]]
}

@test "[convert] zombies-dockerfile-tini" {
  run "$BIN_NAME" convert tests/fixtures/zombies-dockerfile-tini.CHECKS
  echo "output: $output"
  echo "status: $status"
  [[ "$status" -eq 0 ]]
  [[ "$output" == '{"healthchecks":{"web":[{"content":"go","path":"/"}]}}' ]]
}

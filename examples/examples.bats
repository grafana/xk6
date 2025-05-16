#!/usr/bin/env bats

setup_file() {
  export PATH=$BATS_FILE_TMPDIR:$PATH
  cd $BATS_TEST_DIRNAME
  export BASE_DIR=$(git rev-parse --show-toplevel)
  cd $BASE_DIR
  go build -o $BATS_FILE_TMPDIR/xk6 .
  docker build -t grafana/xk6 .
}

setup() {
  cd $BATS_TEST_TMPDIR
}

# bats test_tags=xk6:build
@test 'build.sh' {
  run bash $BATS_TEST_DIRNAME/build.sh
  [ $status -eq 0 ]
  echo "$output" | grep -q "Successful build"
  ./k6 version | grep "github.com/grafana/xk6-example"
  ./k6 version | grep "k6/x/example"
  ./k6 version | grep "github.com/grafana/xk6-output-example"
}

# bats test_tags=xk6:build
@test 'build-with-docker.sh' {
  run bash $BATS_TEST_DIRNAME/build-with-docker.sh
  [ $status -eq 0 ]
  echo "$output" | grep -q "xk6 has now produced a new k6 binary"
  ./k6 version | grep "github.com/grafana/xk6-example"
}

# bats test_tags=xk6:new
@test 'new.sh' {
  run bash $BATS_TEST_DIRNAME/new.sh
  [ $status -eq 0 ]
  test -d xk6-demo
  grep -q "Experimenting with k6 extensions" xk6-demo/README.md
  grep -q "package demo" xk6-demo/*.go
}

# bats test_tags=xk6:lint
@test 'lint.sh' {
  cd $BASE_DIR
  run bash $BATS_TEST_DIRNAME/lint.sh
  [ $status -eq 0 ]
  echo "$output" | grep -q "found \`go.k6.io/xk6\` as go module"
}

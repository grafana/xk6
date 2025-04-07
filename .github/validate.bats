#!/usr/bin/env bats

setup() {
  load helpers
  _common_setup

  cd $BATS_TEST_TMPDIR
}

@test 'build (k6 versions: ${K6_VERSIONS[@]:-latest})' {
  for K6_VERSION in "${K6_VERSIONS[@]}"; do
    run $XK6 build $K6_VERSION --with "${IT_MOD}@${IT_VER}"
    [ $status -eq 0 ]
    echo "$output" | grep -q "xk6 has now produced a new k6 binary"
    ./k6 version | grep -q "${IT_MOD} ${IT_VER}"
  done
}

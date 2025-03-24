#!/usr/bin/env bats

setup() {
  BASEDIR="$(git rev-parse --show-toplevel)"
  cd $BASEDIR

  EXE="$(ls ${BASEDIR}/dist/xk6_linux_$(dpkg --print-architecture)_v*/xk6)"
  IFS=', ' read -r -a K6_VERSIONS <<<"${K6_VERSIONS:-latest}"
  K6_EXTENSION_MODULE=github.com/grafana/xk6-faker
  K6_EXTENSION_VERSION="v0.4.3"

  if [ ! -x "$EXE" ]; then
    echo "    - building snapshot" >&3
    goreleaser build --clean --snapshot --single-target
  fi
}

@test 'build (k6 versions: ${K6_VERSIONS[@]:-latest})' {
  for K6_VERSION in "${K6_VERSIONS[@]}"; do
    [ -f ./k6 ] && rm ./k6 </dev/null
    run $EXE build $K6_VERSION --with "${K6_EXTENSION_MODULE}@${K6_EXTENSION_VERSION}"
    [ $status -eq 0 ]
    echo "$output" | grep -q "xk6 has now produced a new k6 binary"
    ./k6 version | grep -q "${K6_EXTENSION_MODULE} ${K6_EXTENSION_VERSION}"
  done
}

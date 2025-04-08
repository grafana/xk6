_common_setup() {
    BASEDIR=$(dirname $BATS_TEST_DIRNAME)

    IT_MOD=github.com/grafana/xk6-it
    IT_VER=$(_latest_it_version)

    EXT_MOD=github.com/grafana/xk6-it/ext
    EXT_VER=${IT_VER}

    local arch=$(_get_arch)

    XK6=${XK6:-$(echo ${BASEDIR}/it/xk6)}
    if [ ! -x "$XK6" ]; then
        echo "    - building snapshot" >&3
        cd $BASEDIR
        goreleaser build --clean --snapshot --single-target --id xk6
    fi

    XK6_IMAGE=grafana/xk6:latest-${arch}

    IFS=', ' read -r -a K6_VERSIONS <<<"${K6_VERSIONS:-$(_latest_k6_version)}"

    K6_LATEST_VERSION=$(_latest_k6_version)

    export K6=${BATS_TEST_TMPDIR}/k6
}

_latest_k6_version() {
    _get_latest_version "grafana/k6"
}

_latest_it_version() {
    _get_latest_version "grafana/xk6-it"
}

_get_latest_version() {
    local url=$(curl -s -I "https://github.com/$1/releases/latest" | grep -i location)
    local version="${url##*v}"
    version=${version//[[:space:]]/}
    echo -n "v${version}"
}

_get_arch() {
    local arch="$(docker info -f '{{.Architecture}}')"
    case $arch in
    x86_64)
        echo -n "amd64"
        ;;
    arm64)
        echo -n "arm64"
        ;;
    esac
}

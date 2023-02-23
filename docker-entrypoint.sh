#!/bin/sh
set -e

eval "$(fixuid)"

exec /usr/local/bin/xk6 "$@"

#!/bin/sh
set -e

eval "$(fixids -q xk6 xk6)"

exec xk6 "$@"

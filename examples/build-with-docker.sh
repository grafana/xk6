#!/bin/bash

docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/xk6" grafana/k6 build --with github.com/grafana/xk6-faker

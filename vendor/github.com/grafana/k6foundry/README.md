# k6foundry

`k6foundry` is a CLI for building custom k6 binaries with extensions.

## Prerequisites

A Go language tool chain or a properly configured.

## Installation

### Using go toolchain

If you have a go development environment, the installation can be done with the following command:

```
go install github.com/grafana/k6foundry@latest
```

## Usage

### build

The `build` command builds a custom k6 binary with extensions. Multiple extensions with their versions can be specified. The version of k6 can also be specified. If version is omitted, the `latest` version is used. 

The custom binary can target an specific platform, specified as a `os/arch` pair. By default the platform of the `k6foundry` executable is used as target platform.

The following example shows the options for building a custom k6 `v.0.50.0` binary with the latest version of the kubernetes extension and kafka output extension `v0.7.0`.

```
k6foundry build -v v0.50.0 -d github.com/grafana/xk6-kubernetes -d github.com/grafana/xk6-output-kafka@v0.7.0
```

For more examples run

```
k6foundry build --help
```
Use with Docker

The easiest way to use xk6 is via our [Docker image]. This avoids having to setup a local Go environment, and install xk6 manually.

**Linux**

For example, to build a k6 v1.0.0 binary on Linux with the [xk6-faker] extension:

    docker run --rm -it -u "$(id -u):$(id -g)" -v "${PWD}:/xk6" grafana/xk6 build v1.0.0 \
      --with github.com/grafana/xk6-faker

This would create a `k6` binary in the current working directory.

Note the use of the `-u` (user) option to specify the user and group IDs of the account on the host machine. This is important for the `k6` file to have the same file permissions as the host user.

The `-v` (volume) option is also required to mount the current working directory inside the container, so that the `k6` binary can be written to it.

Note that if you're using SELinux, you might need to add `:z` to the `--volume` option to avoid permission errors. E.g. `-v "${PWD}:/xk6:z"`.

**macOS**

On macOS you will need to use `--os darwin` flag to build a macOS binary.

    docker run --rm -it -u "$(id -u):$(id -g)" -v "${PWD}:/xk6" grafana/xk6 build --os darwin v1.0.0 \
      --with github.com/grafana/xk6-faker

**Windows**

On Windows you can either build a native Windows binary, or, if you're using WSL2, a Linux binary you can use in WSL2.

For the native Windows binary if you're using PowerShell:

    docker run --rm -it -u "$(id -u):$(id -g)" -v "${PWD}:/xk6" grafana/xk6 build --os windows v1.0.0 `
      --with github.com/grafana/xk6-faker --output k6.exe 

For the native Windows binary if you're using cmd.exe:

    docker run --rm -it -v "%cd%:/xk6" grafana/xk6 build --os windows v1.0.0 ^
      --with github.com/grafana/xk6-faker --output k6.exe

For the Linux binary on WSL2, you can use the same command as for Linux.

**Tags**

Docker images can be used with major version, minor version, and specific version tags.

For example, let's say `1.2.3` is the latest xk6 Docker image version.

- the latest release of major version `1` is available using the `1` tag:

      docker run --rm -it -u "$(id -u):$(id -g)" -v "${PWD}:/xk6" grafana/xk6:1

- the latest release of minor version `1.2` is available using the `1.2` tag:

      docker run --rm -it -u "$(id -u):$(id -g)" -v "${PWD}:/xk6" grafana/xk6:1.2

- of course version `1.2.3` is still available using the `v1.2.3` tag:

      docker run --rm -it -u "$(id -u):$(id -g)" -v "${PWD}:/xk6" grafana/xk6:1.2.3

- the latest release is still available using the `latest` tag:

      docker run --rm -it -u "$(id -u):$(id -g)" -v "${PWD}:/xk6" grafana/xk6:latest

[Docker image]: https://hub.docker.com/r/grafana/xk6
[xk6-faker]: https://github.com/grafana/xk6-faker

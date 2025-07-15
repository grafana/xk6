# grafana/xk6

This is the official Docker image for **xk6**, the command-line tool for building custom [k6](https://github.com/grafana/k6) binaries with extensions.

This image allows you to use `xk6` without needing to set up a local Go development environment.

-----

## Usage

To use the image, mount your current working directory to the `/xk6` directory inside the container. To build for a specific operating system, set the `GOOS` environment variable.

The most common command is `xk6 build`. You can also use other `xk6` commands like `xk6 run` to execute a test with a custom binary, or `xk6 new` to scaffold a new k6 extension project.

The following examples build a k6 binary in your current directory that includes the `xk6-faker` extension.

### Linux

To build for **Linux** (the default target inside the container):

```shell
docker run --rm -it -u "$(id -u):$(id -g)" -v "$(pwd):/xk6" \
  grafana/xk6 build --with github.com/grafana/xk6-faker
```

### macOS

To build for **macOS**, use the `--os darwin` flag.

```shell
docker run --rm -it -u "$(id -u):$(id -g)" -v "$(pwd):/xk6" \
  grafana/xk6 build --os darwin --with github.com/grafana/xk6-faker
```

### Windows (PowerShell)

To build for **Windows**, use the `--os windows` flag.

```powershell
docker run --rm -it -v "${pwd}:/xk6" \
  grafana/xk6 build --os windows --with github.com/grafana/xk6-faker
```

### Windows (Command Prompt)

To build for **Windows**, use the `--os windows` flag.

```cmd
docker run --rm -it -v "%cd%:/xk6" \
  grafana/xk6 build --os windows --with github.com/grafana/xk6-faker
```

-----

## Supported Tags

This image uses the official `golang` image as its base. Tags are available in several formats to provide flexibility:

  * **`v1.1.0`**, **`v1.0.0`**, etc.: The most specific tags, corresponding to an exact `xk6` release.
  * **`v1.1`**: Points to the latest patch release within the `v1.1` minor series.
  * **`v1`**: Points to the latest minor and patch release within the `v1` major series.
  * **`latest`**: Points to the most recent stable release of `xk6`.

-----

## Multi-Arch Support

This is a **multi-arch image**, providing builds for both **`linux/amd64`** (for standard x86 systems) and **`linux/arm64`** (for ARM-based systems like Apple Silicon).

When you pull a standard tag (e.g., `grafana/xk6:latest`), Docker automatically selects the correct image for your machine's architecture.

If you need to pull an image for a specific architecture explicitly, you can do so by appending the architecture to the tag:

  * `v1.1.0-amd64`
  * `latest-arm64`

-----

## More Information

For more advanced usage, full documentation, and information about `xk6`, please visit the official [grafana/xk6 GitHub repository](https://github.com/grafana/xk6).

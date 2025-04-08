<!-- #region cli -->
# xk6

**k6 extension development toolbox**



### Main features

- [ ] Create new extension skeleton (project scaffolding)
- [x] Build k6 with extensions
- [x] Run k6 with extensions
- [ ] Check the extension for compliance (lint)
- [ ] Provide reusable GitHub workflows
- [x] Distribute xk6 as a Dev Container Feature

### Use with Docker

The easiest way to use xk6 is via our [Docker image]. This avoids having to setup a local Go environment, and install xk6 manually.

**Linux**

For example, to build a k6 v0.58.0 binary on Linux with the [xk6-faker] extension:

    docker run --rm -it -u "$(id -u):$(id -g)" -v "${PWD}:/xk6" grafana/xk6 build v0.58.0 \
      --with github.com/grafana/xk6-faker

This would create a `k6` binary in the current working directory.

Note the use of the `-u` (user) option to specify the user and group IDs of the account on the host machine. This is important for the `k6` file to have the same file permissions as the host user.

The `-v` (volume) option is also required to mount the current working directory inside the container, so that the `k6` binary can be written to it.

Note that if you're using SELinux, you might need to add `:z` to the `--volume` option to avoid permission errors. E.g. `-v "${PWD}:/xk6:z"`.

**macOS**

On macOS you will need to use `--os darwin` flag to build a macOS binary.

    docker run --rm -it -u "$(id -u):$(id -g)" -v "${PWD}:/xk6" grafana/xk6 build --os darwin v0.58.0 \
      --with github.com/grafana/xk6-faker

**Windows**

On Windows you can either build a native Windows binary, or, if you're using WSL2, a Linux binary you can use in WSL2.

For the native Windows binary if you're using PowerShell:

    docker run --rm -it -u "$(id -u):$(id -g)" -v "${PWD}:/xk6" grafana/xk6 build --os windows v0.58.0 `
      --with github.com/grafana/xk6-faker --output k6.exe 

For the native Windows binary if you're using cmd.exe:

    docker run --rm -it -v "%cd%:/xk6" grafana/xk6 build --os windows v0.58.0 ^
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

### Local Installation

Precompiled binaries can be downloaded and installed from the [Releases] page.

**Prerequisites**

A [stable version] of the Go toolkit must be installed.

The xk6 tool can also be installed using the `go install` command.

    go install go.k6.io/xk6/xk6@latest

This will install the `xk6` binary in `$GOPATH/bin` directory.

[Releases]: https://github.com/grafana/xk6/releases
[stable version]: https://go.dev/dl/

## Commands

* [xk6 version](#xk6-version)	 - Display version information
* [xk6 build](#xk6-build)	 - Build a custom k6 executable
* [xk6 run](#xk6-run)	 - Execute the run command with the custom k6

---

# xk6 version

Display version information

## Synopsis

The version is printed to standard output in the following format:

    xk6 version XXX

XXX is the semantic version of xk6, without the v prefix.

## Usage

```bash
xk6 version [flags]
```

## Global Flags

```
  -h, --help      Help about any command 
  -q, --quiet     Suppress output
  -v, --verbose   Verbose output
```

## SEE ALSO

* [xk6](#xk6)	 - k6 extension development toolbox

---

# xk6 build

Build a custom k6 executable

## Synopsis

This command can be used to build custom k6 executables with or without extensions.

The target platform (operating system, architecture) can be specified with flags or environment variables.

The k6 version to be used and the k6 repository (for forks) can be specified with flags or environment variables.

**Precedence**

If a setting can be specified with both a flag and an environment variable, the flag takes precedence.

**Extensions**

The `--with` flag can be used to specify one or more extensions to be included. Extensions can be referenced with the go module path, optionally followed by a version specification. In the case of a fork, the path of the forked go module can be specified as replacement.

**Fork**

The `--replace` flag can be used to specify a replacement for any go module. This allows forks to be used instead of extension dependencies.

A k6 fork can be specified with the `--k6-repo` flag (or the `K6_REPO` environment variable).

## Usage

```bash
xk6 build [flags] [k6-version]
```

## Flags

```
  -o, --output string                         Output filename (default "./k6")
      --with module[@version][=replacement]   Add one or more k6 extensions with Go module path
      --replace module=replacement            Replace one or more Go modules
  -k, --k6-version string                     The k6 version to use for build (default "latest")
      --k6-repo string                        The k6 repository to use for the build (default "go.k6.io/k6")
      --os string                             The target operating system (default "linux")
      --arch string                           The target architecture (default "amd64")
      --arm string                            The target ARM version
      --skip-cleanup int[=1]                  Keep the temporary build directory
      --race-detector int[=1]                 Enable/disable race detector
      --cgo int[=1]                           Enable/disable cgo
      --build-flags stringArray               Specify Go build flags (default [-trimpath,-ldflags=-s -w])
```

## Global Flags

```
  -h, --help      Help about any command 
  -q, --quiet     Suppress output
  -v, --verbose   Verbose output
```

## Environment

```
  K6_VERSION             The k6 version to use for build
  XK6_K6_REPO            The k6 repository to use for the build
  GOOS                   The target operating system
  GOARCH                 The target architecture
  GOARM                  The target ARM version
  XK6_SKIP_CLEANUP       Keep the temporary build directory
  XK6_RACE_DETECTOR      Enable/disable race detector
  CGO                    Enable/disable cgo
  XK6_BUILD_FLAGS        Specify Go build flags
```

## SEE ALSO

* [xk6](#xk6)	 - k6 extension development toolbox

---

# xk6 run

Execute the run command with the custom k6

## Synopsis

This is a useful command when developing the k6 extension. After modifying the source code of the extension, a k6 test script can simply be run without building the k6 executable.

Under the hood, the command builds a k6 executable into a temporary directory and runs it with the arguments. The usual flags for the build command can be used.

Two dashes are used to indicate that the following flags are no longer the flags of the `xk6 run` command but the flags of the `k6 run` command.

## Usage

```bash
xk6 run [flags] [--] [k6-flags] script
```

## Flags

```
      --with module[@version][=replacement]   Add one or more k6 extensions with Go module path
      --replace module=replacement            Replace one or more Go modules
  -k, --k6-version string                     The k6 version to use for build (default "latest")
      --k6-repo string                        The k6 repository to use for the build (default "go.k6.io/k6")
      --os string                             The target operating system (default "linux")
      --arch string                           The target architecture (default "amd64")
      --arm string                            The target ARM version
      --skip-cleanup int[=1]                  Keep the temporary build directory
      --race-detector int[=1]                 Enable/disable race detector
      --cgo int[=1]                           Enable/disable cgo
      --build-flags stringArray               Specify Go build flags (default [-trimpath,-ldflags=-s -w])
```

## Global Flags

```
  -h, --help      Help about any command 
  -q, --quiet     Suppress output
  -v, --verbose   Verbose output
```

## Environment

```
  K6_VERSION             The k6 version to use for build
  XK6_K6_REPO            The k6 repository to use for the build
  GOOS                   The target operating system
  GOARCH                 The target architecture
  GOARM                  The target ARM version
  XK6_SKIP_CLEANUP       Keep the temporary build directory
  XK6_RACE_DETECTOR      Enable/disable race detector
  CGO                    Enable/disable cgo
  XK6_BUILD_FLAGS        Specify Go build flags
```

## SEE ALSO

* [xk6](#xk6)	 - k6 extension development toolbox

<!-- #endregion cli -->

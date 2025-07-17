<!-- #region cli -->
# xk6

**k6 extension development toolbox**



### Main features

- Create new extension skeleton (project scaffolding)
- Build k6 with extensions
- Run k6 with extensions
- Check the extension for compliance (lint)
- Provide reusable GitHub workflows
- Distribute xk6 as a Dev Container Feature

### Use with Development Containers

Get started developing k6 extensions quickly!

xk6 is now a [Development containers] feature, meaning you can develop without installing any tooling or xk6.

Check out the [k6 extension development quickstart guide] and [k6 extension development tutorial] for details.

[Development containers]: https://containers.dev/
[k6 extension development quickstart guide]: https://github.com/grafana/xk6/wiki/k6-extension-development-quick-start-guide
[k6 extension development tutorial]: https://github.com/grafana/xk6/wiki/k6-extension-development-tutorial

### Use with Docker

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

### Local Installation

Precompiled binaries can be downloaded and installed from the [Releases] page.

**Prerequisites**

A [stable version] of the Go toolkit must be installed.

The xk6 tool can also be installed using the `go install` command.

    go install go.k6.io/xk6@latest

This will install the `xk6` binary in `$GOPATH/bin` directory.

[Releases]: https://github.com/grafana/xk6/releases
[stable version]: https://go.dev/dl/

## Commands

* [xk6 version](#xk6-version)	 - Display version information
* [xk6 new](#xk6-new)	 - Create a new k6 extension
* [xk6 build](#xk6-build)	 - Build a custom k6 executable
* [xk6 run](#xk6-run)	 - Execute the run command with the custom k6
* [xk6 lint](#xk6-lint)	 - Static analyzer for k6 extensions
* [xk6 sync](#xk6-sync)	 - Synchronize dependencies with k6

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

# xk6 new

Create a new k6 extension

## Synopsis

Create and initialize a new k6 extension using one of the predefined templates.

The go module path of the new extension must be passed as an argument.

An optional extension description can be specified as a flag.
The default description is generated from the go module path as follows:
- remote git URL is generated from the go module path
- the description is retrieved from the remote repository manager

An optional go package name can be specified as a flag.
The default go package name is generated from the go module path as follows:
- the last element of the go module path is kept
- the `xk6-output-` and `xk6-` prefixes are removed
- the `-` characters are replaced with `_` characters

A JavaScript type k6 extension will be generated by default.
The extension type can be optionally specified as a flag.

The `grafana/xk6-example` and `grafana/xk6-output-example` GitHub repositories are used as sources for generation.

## Usage

```bash
xk6 new [flags] module
```

## Flags

```
  -t, --type string          The type of template to use (javascript or output)
  -d, --description string   A short, on-sentence description of the extension
  -p, --package string       The go package name for the extension
  -C, --parent-dir string    The parent directory (default ".")
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
  CGO_ENABLED            Enable/disable cgo
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
  CGO_ENABLED            Enable/disable cgo
  XK6_BUILD_FLAGS        Specify Go build flags
```

## SEE ALSO

* [xk6](#xk6)	 - k6 extension development toolbox

---

# xk6 lint

Static analyzer for k6 extensions

## Synopsis

**Linter for k6 extensions**

xk6 lint analyzes the source of the k6 extension and try to build k6 with the extension.

The contents of the source directory are used for analysis. If the directory is a git workdir, it also analyzes the git metadata. The analysis is completely local and does not use external APIs (e.g. repository manager API) or services.

The result of the analysis is compliance expressed as a percentage (`0`-`100`). This value is created as a weighted, normalized value of the scores of each checker. A compliance grade is created from the percentage value (`A`-`F`).

By default, text output is generated. The `--json` flag can be used to generate the result in JSON format.

If the grade is `C` or higher, the command is successful, otherwise it returns an exit code larger than `0`.
This passing grade can be modified using the `--passing` flag.

## Usage

```bash
xk6 lint [flags] [directory]
```

### Checkers

Compliance with the requirements expected of k6 extensions is checked by various compliance checkers. The result of the checks is compliance as a percentage value (`0-100`). This value is created as a weighted, normalized value of the scores of each checker. A compliance grade is created from the percentage value (`A`-`F`, `A` is the best).

- `security` - check for security issues (using the `gosec` tool)
- `vulnerability` - check for vulnerability issues (using the `govulncheck` tool)
- `module` - checks if there is a valid `go.mod`
- `replace` - checks if there is no `replace` directive in `go.mod`
- `readme` - checks if there is a readme file
- `examples` - checks if there are files in the `examples` directory
- `license` - checks whether there is a suitable OSS license
- `git` - checks if the directory is git workdir
- `versions` - checks for semantic versioning git tags
- `build` - checks if the latest k6 version can be built with the extension
- `smoke` - checks if the smoke test script exists and runs successfully (`smoke.js`, `smoke.ts`, `smoke.test.js` or `smoke.test.ts` in the `test`,`tests`, `examples`, `scripts` or in the base directory)
- `types` - checks if the TypeScript API declaration file exists (`index.d.ts` in the `docs`, `api-docs` or the base directory)
- `codeowners` - checks if there is a `CODEOWNERS` file (for official extensions) (in the `.github` or `docs` or in the base directory)

## Flags

```
      --passing A|B|C|D|E|F|G|Z   Set lowest passing grade (default C)
      --official                  Enable extra checks for official extensions
  -o, --out string                Write output to file instead of stdout
      --json                      Generate JSON output
  -c, --compact                   Compact instead of pretty-printed JSON output
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

# xk6 sync

Synchronize dependencies with k6

## Synopsis

Synchronizes the versions of dependencies in `go.mod` with those used in the k6 project. Dependencies not found in k6's `go.mod` remain unchanged. Future updates may include synchronization of other files.

The purpose of this subcommand is to avoid dependency conflicts when building the extension with k6 (and other extensions).

It is recommended to keep dependencies in common with k6 core in the same version k6 core uses. This guarantees binary compatibility of the JS runtime, and ensures uses will not have to face unforeseen build-time errors when compiling several extensions together with xk6.

## Usage

```bash
xk6 sync [flags]
```

## Flags

```
  -k, --k6-version string   The k6 version to use for synchronization (default from go.mod)
  -n, --dry-run             Do not make any changes, only log them
  -o, --out string          Write output to file instead of stdout
      --json                Generate JSON output
  -c, --compact             Compact instead of pretty-printed JSON output
  -m, --markdown            Generate Markdown output
```

## Global Flags

```
  -h, --help      Help about any command 
  -q, --quiet     Suppress output
  -v, --verbose   Verbose output
```

## SEE ALSO

* [xk6](#xk6)	 - k6 extension development toolbox

<!-- #endregion cli -->

---

> This project originally forked from the [xcaddy](https://github.com/caddyserver/xcaddy) project. **Thank you!**

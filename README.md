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

### Building private extensions

To build an `xk6` extension from a private Git repository, you need to configure your environment to handle authentication.

**Core Prerequisite**

First, you must set the **`GOPRIVATE`** environment variable. This tells the Go compiler to bypass the standard Go proxy for your repository, allowing it to access the private module directly.

    export GOPRIVATE=github.com/owner/repo

**Method 1: Using SSH**

To handle authentication in non-interactive environments like CI/CD pipelines, configure Git to use the **SSH protocol** instead of HTTPS. This allows for authentication with an SSH key. This command globally configures Git to rewrite any `https://github.com/` URLs to `ssh://git@github.com/`.

    git config --global url.ssh://git@github.com/.insteadOf https://github.com/

**Method 2: Using GitHub CLI**

An alternative to using SSH is to leverage the **GitHub CLI** as a Git credential helper. In this case, Git will still access the repository over HTTPS, but it will use the GitHub CLI to handle the authentication process, eliminating the need to manually enter a password.

    git config --global --add 'credential.https://github.com.helper' '!gh auth git-credential'

## Commands

* [xk6 version](#xk6-version)	 - Display version information
* [xk6 new](#xk6-new)	 - Create a new k6 extension
* [xk6 build](#xk6-build)	 - Build a custom k6 executable
* [xk6 run](#xk6-run)	 - Execute the run command with the custom k6
* [xk6 lint](#xk6-lint)	 - Analyze k6 extension compliance
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

Analyze k6 extension compliance

## Synopsis

Validate k6 extension source code against quality, security, and compatibility standards.
Performs static analysis, builds the extension with k6, and checks compliance requirements.

Use presets to run predefined sets of checks, or customize with individual checkers.
The analysis is performed locally using the source directory contents and Git metadata.

Exit Codes:
  - `0`   All checks passed
  - `>0`  One or more checks failed

## Usage

```bash
xk6 lint [flags] [directory]
```

## Examples

```
# Analyze current directory with default preset
xk6 lint

# Use strict preset for production validation
xk6 lint --preset strict

# Add smoke and examples checks to default preset
xk6 lint --enable smoke,examples

# Run only security checks
xk6 lint --enable-only security,vulnerability  

```

### Available Checks

The following checks are available for use with the `xk6 lint` command.

**`security`**

Performs static security analysis on Go source code using the `gosec` tool to identify potential security vulnerabilities, insecure coding patterns, and compliance violations.

_Security vulnerabilities in extensions can compromise the entire k6 testing environment and potentially expose sensitive data or system resources. Early detection of security flaws through static analysis helps maintain the integrity of the k6 ecosystem and protects users from malicious or poorly secured extensions._

Resolution

Install `gosec` with `go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest`, then run `gosec ./...` to scan your codebase. Address all HIGH and MEDIUM severity findings by following secure coding practices, input validation, and proper error handling. Consider adding `// #gosec` comments only for verified false positives with clear justification.

**`vulnerability`**

Scans for known security vulnerabilities in Go modules and their dependencies using the official `govulncheck` tool from the Go security team.

_Third-party dependencies often contain discovered vulnerabilities that could be exploited in production environments. This check ensures that extensions don't introduce known security risks through outdated or vulnerable dependencies, maintaining the security posture of k6 installations._

Resolution

Install `govulncheck` with `go install golang.org/x/vuln/cmd/govulncheck@latest`, then run `govulncheck ./...` to scan for vulnerabilities. Update vulnerable dependencies to patched versions using `go get -u package@version`. If no patch is available, consider alternative packages or implement additional security measures.

**`module`**

Validates the presence and structure of a `go.mod` file, ensuring proper module declaration, Go version compatibility, and dependency specifications.

_A properly configured `go.mod` file is fundamental for Go module system functionality, enabling reproducible builds, version management, and dependency resolution. Without it, the extension cannot be properly integrated into the k6 build process or distributed through Go's module system._

Resolution

Create a `go.mod` file in the extension root using `go mod init github.com/your-org/your-extension`, ensuring the Go version is specified as `go 1.23` (or appropriate minimum version). Run `go mod tidy` to populate dependencies and remove unused ones, then verify the module path matches your repository structure.

**`replace`**

Detects and flags any `replace` directives in the `go.mod` file that could cause dependency resolution issues or prevent proper extension distribution.

_Replace directives create local overrides that only work in the development environment and break when the extension is built by xk6 or distributed to users. They can mask dependency conflicts, create irreproducible builds, and prevent proper version resolution in the broader Go ecosystem._

Resolution

Remove all `replace` directives from `go.mod`. If you need to use a fork or modified dependency, publish it as a proper Go module with a different import path. For local development, consider using `go work` workspaces instead of replace directives, or contribute fixes upstream to the original repository.

**`readme`**

Verifies the existence of a README file in standard formats (Markdown, text, AsciiDoc, etc.) that provides essential information about the extension.

_A comprehensive README serves as the primary documentation entry point, helping users understand the extension's purpose, installation process, usage examples, and contribution guidelines. It significantly impacts adoption rates and reduces support burden by providing self-service information for common questions._

Resolution

Create a `README.md` file in the extension root directory containing extension description and purpose, installation instructions via xk6, and usage examples with sample k6 scripts. Include API documentation or links to detailed docs, contributing guidelines and development setup, and license information and acknowledgments.

**`license`**

Validates that the extension includes a recognized open-source license file compatible with the k6 ecosystem and Go module distribution requirements.

_A clear license is legally required for code distribution and defines usage rights for users, contributors, and organizations. Without proper licensing, extensions cannot be safely used in commercial environments or contributed to by the community. Accepted licenses ensure compatibility with k6's Apache 2.0 license._

Resolution

Add a `LICENSE` file to the repository root with one of the approved licenses: MIT (recommended for maximum compatibility), Apache-2.0 (best for corporate environments), BSD-2-Clause or BSD-3-Clause, or GPL-3.0, LGPL-3.0, or AGPL-3.0 (for copyleft requirements).

**`git`**

Verifies that the extension directory is a valid Git repository with proper version control initialization and configuration.

_Git version control is essential for extension development, enabling change tracking, collaboration, release management, and integration with Go's module system which relies on Git tags for versioning. Extensions without Git cannot be properly distributed or versioned through standard Go tooling._

Resolution

Initialize Git in the extension directory with `git init`, add a `.gitignore` file appropriate for the extension, then stage and commit all extension files using `git add . && git commit -m "Initial commit"`. Consider setting up a remote repository on GitHub, GitLab, or similar platform for collaboration and distribution.

**`versions`**

Validates the presence of proper semantic versioning Git tags following the vMAJOR.MINOR.PATCH format required by Go modules and xk6.

_Semantic versioning tags are critical for Go module resolution, allowing users to specify version constraints and enabling automatic dependency management. Proper versioning communicates API compatibility, helps users understand upgrade risks, and enables tools like Dependabot to manage updates automatically._

Resolution

Create an initial release tag using `git tag v0.1.0 && git push origin v0.1.0`. For future releases, increment versions appropriately: PATCH (v1.0.1) for bug fixes with no API changes, MINOR (v1.1.0) for new features that are backward compatible, and MAJOR (v2.0.0) for breaking changes or API modifications. Always follow semantic versioning principles for predictable dependency management.

**`build`**

Performs a complete build test of the extension using xk6 with the latest stable k6 version to verify compilation and linking compatibility.

_Build compatibility is essential for user adoption and long-term maintainability. This check catches compilation errors, API compatibility issues, and dependency conflicts that would prevent users from successfully building custom k6 binaries with the extension, ensuring a smooth user experience._

Resolution

Test the build locally using `xk6 build --with github.com/your-org/your-extension@latest`, then fix any compilation errors, missing imports, or API incompatibilities. Ensure your extension properly implements required k6 extension interfaces and update dependencies if needed using `go get -u && go mod tidy`.

**`smoke`**

Locates and executes a smoke test script to verify basic extension functionality works correctly in a real k6 runtime environment.

_Smoke tests provide essential validation that the extension's core functionality operates as expected when loaded into k6. They catch runtime errors, API mismatches, and integration issues that static analysis cannot detect, serving as the minimum viable test to ensure the extension actually works for end users._

Resolution

Create a smoke test file as `smoke.js` or `smoke.ts` in root, `test/`, `tests/`, or `examples/` directory, including basic functionality tests that import the extension and call main functions. Ensure the test runs without errors when executed with your custom k6 build.

**`examples`**

Ensures the presence of an `examples/` directory containing practical k6 scripts that demonstrate the extension's functionality and usage patterns.

_Example scripts are crucial for user onboarding and adoption, providing immediate practical value and reducing the learning curve. They serve as living documentation, showing real-world usage patterns and helping users quickly understand how to integrate the extension into their testing workflows._

Resolution

Create an `examples/` directory with multiple k6 JavaScript/TypeScript files including a basic usage example showing core functionality, advanced example demonstrating complex features, and integration examples with other k6 features. Include comments explaining key concepts and parameters. Add a README.md in examples/ explaining how to run each script.

**`types`**

Validates the presence of TypeScript declaration files (`index.d.ts`) that define the extension's API surface and enable type-safe usage in TypeScript k6 scripts.

_TypeScript declarations significantly improve developer experience by providing IDE autocompletion, type checking, and inline documentation. As k6 increasingly supports TypeScript, providing accurate type definitions becomes essential for extension adoption and proper integration with modern development workflows._

Resolution

Create an `index.d.ts` file in the root, `docs/`, or `api-docs/` directory defining all exported functions, classes, and interfaces with parameter types and return types. Include JSDoc comments for function documentation and proper module declarations matching your extension's import path.

**`codeowners`**

Validates the existence of a `CODEOWNERS` file that defines maintainership responsibilities and automated review assignments for different parts of the codebase.

_Code ownership is critical for maintaining extension quality and ensuring timely responses to issues and pull requests. CODEOWNERS enables automatic reviewer assignment, helps contributors identify the right people for questions, and establishes clear accountability for different components of the extension._

Resolution

Create a `CODEOWNERS` file in `.github/`, `docs/`, or the repository root defining global owners (`* @username @team`) and specifying path-based ownership (`docs/ @doc-team`). Include email contacts for critical components and use GitHub teams when possible for better maintainability. Ensure all specified owners have appropriate repository permissions.

### Available Presets

The following presets are available for use with the `xk6 lint` command.

**`all`**

Comprehensive preset that includes every available check in the xk6 linting system. Serves as a complete reference for all possible compliance checks and provides maximum validation coverage for development and testing purposes.

Included Checks:
  - `security`
  - `vulnerability`
  - `module`
  - `replace`
  - `readme`
  - `license`
  - `git`
  - `versions`
  - `build`
  - `smoke`
  - `examples`
  - `types`
  - `codeowners`

**`loose`**

Minimal preset focusing on essential quality and security compliance checks. Designed for development environments and initial extension development phases. Provides basic compliance requirements without restrictive validation that slows development cycles. This is the default preset.

Included Checks:
  - `security`
  - `vulnerability`
  - `module`
  - `readme`
  - `license`
  - `git`
  - `versions`
  - `build`

**`strict`**

Comprehensive preset for production-ready extensions, including all compliance checks except those reserved for official Grafana extensions (such as codeowners validation). Designed for third-party extensions that require high quality standards before release.

Included Checks:
  - `security`
  - `vulnerability`
  - `module`
  - `replace`
  - `readme`
  - `license`
  - `git`
  - `versions`
  - `build`
  - `smoke`
  - `examples`
  - `types`

**`private`**

Lightweight preset designed for private or internal extension development. Focuses on core security and functionality compliance while omitting documentation and public-facing requirements such as README formatting and licensing compliance.

Included Checks:
  - `security`
  - `vulnerability`
  - `module`
  - `git`

**`community`**

Balanced preset tailored for community-contributed extension development. Includes essential quality, security, and documentation compliance to ensure extensions meet community standards while remaining accessible to contributors.

Included Checks:
  - `security`
  - `vulnerability`
  - `module`
  - `readme`
  - `license`
  - `git`
  - `versions`
  - `build`

**`official`**

Most stringent preset for official Grafana-maintained extension development. Enforces the highest quality standards including code ownership compliance, comprehensive testing requirements, and complete documentation compliance.

Included Checks:
  - `security`
  - `vulnerability`
  - `module`
  - `replace`
  - `readme`
  - `license`
  - `git`
  - `versions`
  - `build`
  - `smoke`
  - `examples`
  - `types`
  - `codeowners`

## Flags

```
  -o, --out string             Write output to file instead of stdout
      --json                   Generate JSON output
  -c, --compact                Compact instead of pretty-printed JSON output
  -p, --preset preset          Check preset to use (default: loose) (default loose)
      --enable checkers        Enable additional checks (comma-separated list)
      --disable checkers       Disable specific checks (comma-separated list)
      --enable-only checkers   Enable only specified checks, ignoring preset (comma-separated list)
  -k, --k6-version string      The k6 version to use for build (default "latest")
      --k6-repo string         The k6 repository to use for the build (default "go.k6.io/k6")
```

## Global Flags

```
  -h, --help      Help about any command 
  -q, --quiet     Suppress output
  -v, --verbose   Verbose output
```

## Environment

```
  XK6_LINT_PRESET           Check preset to use (default: loose)
  XK6_LINT_ENABLE           Enable additional checks (comma-separated list)
  XK6_LINT_DISABLE          Disable specific checks (comma-separated list)
  XK6_LINT_ENABLE_ONLY      Enable only specified checks, ignoring preset (comma-separated list)
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

# xk6 workflows

## Repository workflows

The following workflows are used to maintain the **xk6 repository** itself. These workflows are not designed to be reusable, and are subject to change at any time without notice.

### Validate

The **Validate** ([`validate.yml`](../../.github/workflows/validate.yml)) workflow validates the source code change in case of a push or pull request to the default branch. This workflow calls the [Tooling Validate](#tooling-validate) reusable workflow with appropriate parameters. The parameters can be configured in [GitHub repository variables](https://github.com/grafana/xk6/settings/variables/actions) and [GitHub repository secrets](https://github.com/grafana/xk6/settings/secrets/actions).

The [`validate.bats`](../../.github/validate.bats) script is passed as the integration test. The test builds a `k6` with a specific extension using the k6 versions specified in the `K6_VERSIONS` repository variable. After a successful build, it runs the built k6 with the `version` command and checks if the specific extension is included in the output.

**triggers**

```yaml file=../../.github/workflows/validate.yml region=triggers
  workflow_dispatch:
  push:
    branches: ["main", "master"]
  pull_request:
    branches: ["main", "master"]
```

**inputs**

```yaml file=../../.github/workflows/validate.yml region=inputs
      go-version: "1.24.x"
      go-versions: '["1.23.x","1.24.x"]'
      golangci-lint-version: "v2.1.2"
      goreleaser-version: "2.8.2"
      platforms: '["ubuntu-latest", "windows-latest", "macos-latest"]'
      k6-versions: '["v1.0.0","v0.57.0"]'
      bats: .github/validate.bats
```

### Release

The **Release** ([`release.yml`](../../.github/workflows/release.yml)) workflow generates and publishes the release artifacts in case of version tag creation. This workflow calls the [Tooling Release](#tooling-release) reusable workflow with appropriate parameters. The parameters can be configured in GitHub repository variables and GitHub repository secrets.

The [`release.bats`](../../.github/release.bats) script is passed as the integration test. One test case builds a k6 using xk6 with a specific extension using the k6 versions specified in the `K6_VERSIONS` repository variable. After a successful build, it runs the built k6 with the `version` command and checks if the specific extension is included in the output. The other test case does the same thing, but uses the xk6 Docker image to build the k6.

**triggers**

```yaml file=../../.github/workflows/release.yml region=triggers
  push:
    tags: ["v*.*.*"]
```

**inputs**

```yaml file=../../.github/workflows/release.yml region=inputs
      go-version: "1.24.x"
      goreleaser-version: "2.8.2"
      k6-versions: '["v1.0.0","v0.57.0"]'
      bats: ./.github/release.bats
```

## Tooling workflows

The following workflows are used to maintain *tools related to the development of k6 extensions*. These workflows are designed to be **reusable**.

Input parameters will change in a backwards compatible way if possible, but it is strongly recommended to use workflows with a specific version tag.

### Tooling Validate

The **Tooling Validate** ([`tooling-validate.yml`](../../.github/workflows/tooling-validate.yml)) reusable workflow validates the source code modification.

![tooling-validate-dark](tooling-validate-dark.png#gh-dark-mode-only)
![tooling-validate-light](tooling-validate-light.png#gh-light-mode-only)

**Jobs**

- **Config**: Checking the input parameters and generating various configuration properties. Creating a summary report from the configuration properties.

- **Security**: Basic security and vulnerability checks using the [gosec](https://github.com/securego/gosec) and [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) tools. If the check fails, the workflow will stop with an error.

- **DevContainer**: If a [Development Container](https://containers.dev/) configuration exists, it validates its parameters based on the workflow input parameters. In case of incorrect configuration, it indicates an error but does not stop the workflow.

- **Lint**: Static analysis of the source code using the [golangci-lint](https://golangci-lint.run/) tool.  It uses the golangci-lint configuration found in the repository, or if it is missing, the following arguments:
  ```
  --no-config --presets bugs --enable gofmt
  ```
  In case of an error, the workflow will stop with an error.

- **Smoke**: Run short Go tests (`-short` flag) on single platform and single go version. In case of an error, the workflow will stop with an error. The role of this job is to quickly stop the workflow in case of trivial test errors, preventing slower tests from running on multiple platforms using multiple go versions.

- **Test**: Run Go tests on multiple platforms using multiple go versions. The execution will be done using a matrix strategy, the number of jobs run is the number of go versions multiplied by the number of platforms. The workflow run does not stop if the first job fails, all jobs in the matrix are executed. If any job fails, the workflow will stop with an error. A typical input contains two go versions and three platforms (`ubuntu-latest`, `windows-latest`, `macos-latest`) which is 6 jobs in the matrix.

- **Build**: Create builds using [GoReleaser](https://goreleaser.com/) tool on multiple platforms using multiple go versions. The build will be done with a matrix strategy, the number of jobs run is the number of go versions multiplied by the number of platforms. The build does not stop on the first job failure, all jobs in the matrix are executed. If any job fails, the workflow will stop. The typical input contains two go versions and three platforms (`ubuntu-latest`, `windows-latest`, `macos-latest`) which is 6 jobs in the matrix.
  The build is done with the following GoReleaser arguments:
  ```yaml file=../../.github/workflows/tooling-validate.yml region=build-args
            args: build --clean --snapshot --single-target
  ```

  [Bats](https://github.com/bats-core/bats-core) (Bash Automated Testing System) scripts can be specified as integration tests. These will be executed in jobs running on `Linux` operating systems (with all go versions specified in the input). If any integration test script fails, the workflow will stop with an error.

- **Report**: After a successful build and test job run, if the workflow is executed on the default branch and the [Codecov](https://codecov.io/) token secret is included in the input, the test coverage data will be uploaded and the [Go Report Card](https://goreportcard.com/) will be created.

**Secrets**

```yaml file=../../.github/workflows/tooling-validate.yml region=secrets
      codecov-token:
        description: Token to be used to upload test coverage data to Codecov.
        required: false
```

**Inputs**

```yaml file=../../.github/workflows/tooling-validate.yml region=inputs
      go-version:
        description: The go version to use for the build.
        required: true
        type: string
      go-versions:
        description: The go versions to use for running the tests. JSON string array (e.g. ["1.24.x", "1.23.x"])
        required: true
        type: string
      platforms:
        description: Platforms to be used to run the tests. JSON string array (e.g. ["ubuntu-latest","macos-latest"])
        required: true
        type: string
      golangci-lint-version:
        description: The golangci-lint version to use for static analysis.
        required: true
        type: string
      goreleaser-version:
        description: The version of GoReleaser to use for builds and releases.
        required: true
        type: string
      k6-versions:
        description: The k6 versions to be used for integration tests. JSON string array (e.g. ["v0.57.0","v0.56.0"])
        required: false
        default: '["latest"]'
        type: string
      bats:
        description: The bats scripts to use for integration testing. Space-separated file names or patterns.
        type: string
        required: false
```

### Tooling Release

The **Tooling Release** ([`tooling-release.yml`](../../.github/workflows/tooling-release.yml)) reusable workflow generates and publishes release artifacts (including Docker images).

![tooling-release-dark](tooling-release-dark.png#gh-dark-mode-only)
![tooling-release-light](tooling-release-light.png#gh-light-mode-only)

**Jobs**

- **Config**: Checking the input parameters and generating various configuration properties. Creating a summary report from the configuration properties.

- **Security**: Basic security and vulnerability checks using the [gosec](https://github.com/securego/gosec) and [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) tools. If the check fails, the workflow will stop with an error. Last chance to catch security and vulnerability issues before release.

- **Release**: Create and publish release artifacts using [GoReleaser](https://goreleaser.com/) tool. Before creating the final release artifacts, a snapshot build is created. [Bats](https://github.com/bats-core/bats-core) (Bash Automated Testing System) scripts can be specified as integration tests for the snapshot build. These will be executed and if any of them fail, the workflow will stop with an error.

  The created release artifacts will be published to GitHub releases. If the GoReleaser configuration contains a Docker image configuration, the created Docker images will also be published to the appropriate Docker registry. GitHub Packages and Docker Hub repositories are supported.  It is even possible to publish to both registries at the same time.

**Inputs**

```yaml file=../../.github/workflows/tooling-release.yml region=inputs
      go-version:
        description: The go version to use for the build.
        required: true
        type: string
      goreleaser-version:
        description: The version of GoReleaser to use for builds and releases.
        required: true
        type: string
      k6-versions:
        description: The k6 versions to be used for integration tests. JSON string array (e.g. ["v0.57.0","v0.56.0"])
        required: false
        default: '["latest"]'
        type: string
      bats:
        description: The bats scripts to use for integration testing. Space-separated file names or patterns.
        type: string
        required: false
```

---

## Extension workflows

The following workflows are used to *maintain k6 extensions*. These workflows are designed to be **reusable**.

Input parameters will change in a backwards compatible way if possible, but it is strongly recommended to use workflows with a specific version tag.

### Extension Validate

The **Extension Validate** ([`extension-validate.yml`](../../.github/workflows/extension-validate.yml)) reusable workflow validates the source code modification.

![extension-validate-dark](extension-validate-dark.png#gh-dark-mode-only)
![extension-validate-light](extension-validate-light.png#gh-light-mode-only)

**Jobs**

- **Config**: Checking the input parameters and generating various configuration properties. Creating a summary report from the configuration properties.

- **Security**: Basic security and vulnerability checks using the [gosec](https://github.com/securego/gosec) and [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) tools. If the check fails, the workflow will stop with an error.

- **DevContainer**: If a [Development Container](https://containers.dev/) configuration exists, it validates its parameters based on the workflow input parameters. In case of incorrect configuration, it indicates an error but does not stop the workflow.

- **Lint**: Static analysis of the source code using the [golangci-lint](https://golangci-lint.run/) tool.  It uses the golangci-lint configuration found in the repository, or if it is missing, the following arguments:
  ```
  --no-config --presets bugs --enable gofmt
  ```
  In case of an error, the workflow will stop with an error.

- **Smoke**: Run short Go tests (`-short` flag) on single platform and single go version. In case of an error, the workflow will stop with an error. The role of this job is to quickly stop the workflow in case of trivial test errors, preventing slower tests from running on multiple platforms using multiple go versions.

- **Compliance**: Check the extension for compliance with best practices using the `xk6 lint` command. If the compliance level does not reach the value specified in the `passing-grade` input parameter (default `C`), the workflow will fail.

- **Test**: Run Go tests on multiple platforms using multiple go versions. The execution will be done using a matrix strategy, the number of jobs run is the number of go versions multiplied by the number of platforms. The workflow run does not stop if the first job fails, all jobs in the matrix are executed. If any job fails, the workflow will stop with an error. A typical input contains two go versions and three platforms (`ubuntu-latest`, `windows-latest`, `macos-latest`) which is 6 jobs in the matrix.

- **Build**: Create custom k6 builds with extension using [xk6](https://github.com/grafana/xk6/) tool on multiple platforms using multiple go versions. The build will be done with a matrix strategy, the number of jobs run is the number of go versions multiplied by the number of platforms and the number of k6 versions. The build does not stop on the first job failure, all jobs in the matrix are executed. If any job fails, the workflow will stop. The typical input contains two go versions, two k6 versions and three platforms (`ubuntu-latest`, `windows-latest`, `macos-latest`) which is 12 jobs in the matrix.

  [Bats](https://github.com/bats-core/bats-core) (Bash Automated Testing System) scripts can be specified as integration tests. These will be executed in jobs running on `Linux` operating systems (with all go versions specified in the input). If any integration test script fails, the workflow will stop with an error.

- **Report**: After a successful build and test job run, if the workflow is executed on the default branch and the [Codecov](https://codecov.io/) token secret is included in the input, the test coverage data will be uploaded and the [Go Report Card](https://goreportcard.com/) will be created.

**Secrets**

```yaml file=../../.github/workflows/extension-validate.yml region=secrets
      codecov-token:
        description: Token to be used to upload test coverage data to Codecov.
        required: false
```

**Inputs**

```yaml file=../../.github/workflows/extension-validate.yml region=inputs
      go-version:
        description: The go version to use for the build.
        required: true
        type: string
      go-versions:
        description: The go versions to use for running the tests. JSON string array (e.g. ["1.24.x", "1.23.x"])
        required: true
        type: string
      platforms:
        description: Platforms to be used to run the tests. JSON string array (e.g. ["ubuntu-latest","macos-latest"])
        required: true
        type: string
      golangci-lint-version:
        description: The golangci-lint version to use for static analysis.
        required: true
        type: string
      passing-grade:
        description: Passing compliance grade
        type: string
        required: false
        default: C
      public:
        description: Static content directory for GitHub Pages
        type: string
        required: false
        default: public
      k6-versions:
        description: The k6 versions to be used for integration tests. JSON string array (e.g. ["v0.57.0","v0.56.0"])
        required: false
        default: '["latest"]'
        type: string
      xk6-version:
        description: The xk6 version to be used.
        required: true
        type: string
      bats:
        description: The bats scripts to use for integration testing. Space-separated file names or patterns.
        type: string
        required: false
```

### Extension Release

The **Extension Release** ([`extension-release.yml`](../../.github/workflows/extension-release.yml)) reusable workflow generates and publishes release artifacts (including custom k6 executables).

![extension-release-dark](extension-release-dark.png#gh-dark-mode-only)
![extension-release-light](extension-release-light.png#gh-light-mode-only)

**Jobs**

- **Config**: Checking the input parameters and generating various configuration properties. Creating a summary report from the configuration properties.

- **Security**: Basic security and vulnerability checks using the [gosec](https://github.com/securego/gosec) and [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) tools. If the check fails, the workflow will stop with an error. Last chance to catch security and vulnerability issues before release.

- **Build**: Build release artifacts. [Bats](https://github.com/bats-core/bats-core) (Bash Automated Testing System) scripts can be specified as integration tests for the release build. These will be executed and if any of them fail, the workflow will stop with an error.

- **Publish**: Publish release artifacts.

**Inputs**

```yaml file=../../.github/workflows/extension-release.yml region=inputs
      k6-version:
        description: The k6 versions to be used.
        type: string
        required: true
      xk6-version:
        description: The xk6 versions to be used.
        type: string
        required: true
      go-version:
        description: The go version to use for the build.
        type: string
        required: true
      os:
        description: Target GOOS values. JSON string array (e.g. ["linux","windows","darwin"])
        type: string
        required: true
      arch:
        description: Target GOARCH values. JSON string array (e.g. ["amd64","arm64"])
        type: string
        required: true
      with:
        description: List of additional extension modules to be included.
        type: string
        required: false
      cgo:
        description: Enable CGO
        type: boolean
      private:
        description: The repository is private
        type: boolean
      bats:
        description: The bats scripts to use for integration testing. Space-separated file names or patterns.
        type: string
        required: false
```
